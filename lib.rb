require 'fileutils'
require 'set'

# ==============================================================
# UTILITIES:
# functions for logging, tracing, error reporting, coloring text
# ==============================================================

def log(msg, args={})
  puts msg
  max = args.keys.map(&:to_s).max_by(&:length).length
  args.each do |k, v|
    vs = case v
         when Array, Set
           "(#{v.length}) " + v.join(", ")
         else
           v
         end
    puts "\t#{k.to_s.rjust(max)}: #{vs}"
  end
end

def sudo(command)
  system("sudo #{command}")
end


# ==============================================================
# CORE:
# State of the system It should hold all the information we need to build the
# system, packages, files, changes...etc. everything will run inside an instance
# of this class
# ==============================================================
class State
  def apply(block)
    instance_eval &block
  end
end

# Builder takes state as input and modify current system to match this state
class Builder
  @@install_steps = {}

  # Define a new installation step. the passed block will take 1 parameter of State type
  def self.on_install(id, &block)
    @@install_steps[id] = block
  end

  attr_accessor :state
  def initialize(state)
    self.state = state
  end

  # run will rull all steps in their registeration order passing the state to it
  def run
    @@install_steps.each do |_, step|
      state.apply(step)
    end
  end
end

def on_install(id=nil, &block)
  id ||=  caller_locations(1,1).first.to_s
  Builder.on_install(id, &block)
end

def linux(&block)
  s = State.new
  s.apply(block)
  Builder.new(s).run
end


# ==============================================================
# DECLARATIONS:
# Functions the user will run to declare the state of the system
# like packages to be present, files, services, user, group...etc
# ==============================================================

# package command, accumulates packages needs to be installed
def package(*names)
  @packages ||= Set.new
  @packages += names.map(&:to_s)

  # install step to install packages required and remove not required
  on_install do
    # install packages list as is
    names = @packages.join(" ")
    log "Installing packages", packages: @packages
    sudo "pacman --needed -S #{names}" unless @packages.empty?

    # expand groups to packages
    group_packages = Set.new(`pacman --quiet -Sg #{names}`.lines.map(&:strip))

    # full list of packages that should exist on the system
    all = @packages + group_packages

    # actual list on the system
    installed = Set.new(`pacman -Q --quiet --explicit --unrequired --native`.lines.map(&:strip))

    unneeded = installed - all
    log "Removing packages", packages: unneeded
    sudo("pacman -Rs #{unneeded.join(" ")}") unless unneeded.empty?
  end

end

# aur command to install packages from aur
def aur(*names)
  @aurs ||= Set.new
  @aurs += names.map(&:to_s)

  on_install do
    names = @aurs || []
    log "Install AUR packages", packages: names
    cache = "./cache/aur"
    FileUtils.mkdir_p cache
    Dir.chdir cache do
      names.each do |package|
        system("git clone --depth 1 --shallow-submodules https://aur.archlinux.org/#{package}.git")
        Dir.chdir package do
          system("makepkg --syncdeps --install --noconfirm --needed")
        end
      end
    end
  end
end

def timedate(timezone: 'UTC', ntp: true)
  @timedate = {timezone: timezone, ntp: ntp}

  on_install do
    log "Set timedate", timedate: @timedate
    sudo "timedatectl set-timezone #{@timedate[:timezone]}"
    sudo "timedatectl set-ntp #{@timedate[:ntp]}"
  end
end

def service(*names)
  @services ||= Set.new
  @services += names

  on_install do
    log "Enable services", services: @services
    sudo "systemctl enable --now #{@services.join(" ")}"
    # disable all other services
  end
end

def user_service(*names)
  @user_services ||= Set.new
  @user_services += names

  on_install do
    system "systemctl enable --user --now #{@services.join(" ")}"
    # disable all other user services
  end
end

def timer(*names)
  @timers ||= Set.new
  @timers += names

  on_install do
    timers = @timers.map{ |t| "#{t}.timer" }.join(" ")
    sudo "systemctl enable #{timers}"
    # disable all other timers
  end
end

def keyboard(keymap: nil, layout: nil, model: nil, variant: nil, options: nil)
  @keyboard ||= {}
  values = {
    keymap: keymap,
    layout: layout,
    model: model,
    variant: variant,
    options: options
  }.compact
  @keyboard.merge!(values)

  on_install do
    next unless @keyboard[:keymap]

    sudo "localectl set-keymap #{@keyboard[:keymap]}"

    m = @keyboard.to_h.slice(:layout, :model, :variant, :options)
    sudo "localectl set-x11-keymap \"#{m[:layout]}\" \"#{m[:model]}\" \"#{m[:variant]}\" \"#{m[:options]}\""
  end
end

# Users and groups commands
def user(name, groups: [])
end

# create a group and allow/disallow it to sudo
def group(name, sudo: false)
end

# processes commands
def run(command)
  @run ||= Set.new
  @run << command

  on_install do
    @run.each { |cmd| system(cmd) }
  end
end

# Sync src directory with destination directory
def sync(src, dest)
end

# Replace a pattern with replacement string in a file
def replace_line(file, pattern, replacement)
end

# firewall
def firewall(*allow)
  @firewall ||= Set.new
  @firewall += allow

  package :ufw
  service :ufw

  on_install do
    next unless @firewall
    sudo "ufw allow #{@firewall.join(' ')}"
  end
end
