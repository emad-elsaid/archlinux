require 'fileutils'
require 'set'
require 'etc'

Signal.trap("INT") { exit } # Suppress stack trace on Ctrl-C

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

def root?
  Process.uid == Etc.getpwnam('root').uid
end

def sudo(command)
  root? ? system(command) : system("sudo #{command}")
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

  # Adds a step to installation steps, ID identifies the step to make sure same
  # step is not executed twice, in case the ID is nil the caller location will be
  # used the ID, meaning the place where on_install is called from if ran twice
  # it'll be added once
  def on_install(id=nil, &block)
    id ||=  caller_locations(1,1).first.to_s
    @install_steps ||= {}
    @install_steps[id] = block
  end

  # run will rull all steps in their registeration order
  def run_steps
    @install_steps.each do |_, step|
      apply(step)
    end
  end
end

# passed block will run in the context of a State instance and then a builder
# will build this state
def linux(&block)
  s = State.new
  s.apply(block)
  s.run_steps
end


# ==============================================================
# DECLARATIONS:
# Functions the user will run to declare the state of the system
# like packages to be present, files, services, user, group...etc
# ==============================================================

# package command, accumulates packages needs to be installed
def package(*names)
  names.flatten!
  @packages ||= Set.new
  @packages += names.map(&:to_s)

  # install step to install packages required and remove not required
  on_install do
    # install packages list as is
    names = @packages.join(" ")
    log "Installing packages", packages: @packages
    sudo "pacman --noconfirm --needed -S #{names}" unless @packages.empty?

    # expand groups to packages
    group_packages = Set.new(`pacman --quiet -Sg #{names}`.lines.map(&:strip))

    # full list of packages that should exist on the system
    all = @packages + group_packages

    # actual list on the system
    installed = Set.new(`pacman -Q --quiet --explicit --unrequired --native`.lines.map(&:strip))

    unneeded = installed - all
    next if unneeded.empty?

    log "Removing packages", packages: unneeded
    sudo("pacman -Rs #{unneeded.join(" ")}")
  end

end

# aur command to install packages from aur
def aur(*names)
  names.flatten!
  @aurs ||= Set.new
  @aurs += names.map(&:to_s)

  on_install do
    names = @aurs || []
    log "Install AUR packages", packages: names
    cache = "./cache/aur"
    FileUtils.mkdir_p cache
    Dir.chdir cache do
      names.each do |package|
        system("git clone --depth 1 --shallow-submodules https://aur.archlinux.org/#{package}.git") unless Dir.exists?(package)
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
    log "Set timedate", @timedate
    sudo "timedatectl set-timezone #{@timedate[:timezone]}"
    sudo "timedatectl set-ntp #{@timedate[:ntp]}"
  end
end

def service(*names)
  names.flatten!
  @services ||= Set.new
  @services += names

  on_install do
    log "Enable services", services: @services
    if root?
      system "systemctl enable #{@services.join(" ")}"
    else
      system "systemctl enable --user --now #{@services.join(" ")}"
    end
    # disable all other services
  end
end

def timer(*names)
  names.flatten!
  @timers ||= Set.new
  @timers += names

  on_install do
    log "Enable timers", timers: @timers
    timers = @timers.map{ |t| "#{t}.timer" }.join(" ")
    if root?
      sudo "systemctl enable #{timers}"
    else
      system "systemctl enable --user #{timers}"
    end
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
def user(name, groups: [], &block)
  @user ||= {}
  @user[name] ||= {}
  @user[name][:groups] ||= []
  @user[name][:groups] += groups
  @user[name][:state] = State.new
  @user[name][:state].apply(block) if block_given?

  on_install do
    @user.each do |name, conf|
      if groups.empty?
        sudo "useradd --groups #{groups.join(",")} #{name}"
      else
        sudo "useradd #{name}"
      end

      fork do
        currentuser = Etc.getpwnam(name)
        Process::GID.change_privilege(currentuser.gid)
        Process::UID.change_privilege(currentuser.uid)
        ENV['XDG_RUNTIME_DIR'] = "/run/user/#{currentuser.uid}"
        conf[:state].run_steps
      end
      Process.wait
    end
  end
end

# processes commands
def run(command)
  @run ||= Set.new
  @run << command

  on_install do
    @run.each { |cmd| system(cmd) }
  end
end

# Copy src inside dest during install step, if src/. will copy src content to dest
def copy(src, dest)
  @copy ||= []
  @copy << { src: src, dest: dest }

  on_install do
    next unless @copy
    next if @copy.empty?

    @copy.each do |item|
      log "Copying", item
      FileUtils.cp_r item[:src], item[:dest]
    end
  end
end

# Replace a regex pattern with replacement string in a file
def replace(file, pattern, replacement)
  input = File.read(file)
  output = input.gsub(pattern, replacement)
  File.write(file, output)
end

# setup add ufw enable it and allow ports
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
