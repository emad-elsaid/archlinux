# State of the system It should hold all the information we need to build the
# system, packages, files, changes...etc. everything will run inside an instance
# of this class
class State
  def apply(block)
    instance_eval &block
  end
end

# Builder takes state as input and modify current system to match this state
class Builder
  @@install_steps = []

  # Define a new installation step. the passed block will take 1 parameter of State type
  def self.install(&block)
    @@install_steps << block
  end

  attr_accessor :state
  def initialize(state)
    self.state = state
  end

  # run will rull all steps in their registeration order passing the state to it
  def run
    @@install_steps.each do |step|
      state.apply(step)
    end
  end
end

def linux(&block)
  s = State.new
  s.apply(block)
  Builder.new(s).run
end


# package command, accumulates packages needs to be installed
def package(*names)
  @packages ||= []
  @packages += names
end

# install step to install packages required and remove not required
Builder.install do
  # install packages list as is
  names = @packages.join(" ")
  system("pacman --needed -S #{names}") unless @packages.empty?

  # expand groups to packages
  group_packages = `pacman --quiet -Sg #{names}`.lines.map(&:strip)

  # full list of packages that should exist on the system
  all = (@packages + group_packages).uniq

  # actual list on the system
  # TODO this list doesn't include packages that were
  # explicitly installed AND is a dependency at the same time like `linux` which is dependency of `base`
  # or `gimp` which is optional dependency of `alsa-lib`
  installed = `pacman -Q --quiet --explicit --unrequired --native`.lines.map(&:strip)

  unneeded = installed - all
  puts "Removing packages: #{unneeded}"
  system("pacman -Rs #{unneeded.join(" ")}") unless unneeded.empty?
end

# aur command to install packages from aur
def aur(*names)
  @aurs ||= ['yay']
  @aurs += names
end

Builder.install do
  names = @aurs || []
  flags = '--norebuild --noredownload --editmenu=false --diffmenu=false --noconfirm'
  system("yay #{flags} -S #{names.join(" ")}") unless names.empty?

  installed = `pacman -Q --quiet --explicit --unrequired --foreign`.lines.map(&:strip)
  unneeded = installed - names
  puts "Removing packages: #{unneeded}"
  system("pacman -Rs #{unneeded.join(" ")}") unless unneeded.empty?
end

# Systemd

def timedate(timezone: 'UTC', ntp: true)
  @timedate = {timezone: timezone, ntp: ntp}
end

Builder.install do
  system("timedatectl set-timezone #{@timedate[:timezone]}")
  system("timedatectl set-ntp #{@timedate[:ntp]}")
end

def service(*names)
  @services ||= []
  @services += names
end

Builder.install do
  system("systemctl enable --now #{@services.join(" ")}")
  # disable all other services
end

def user_service(*names)
  @user_services ||= []
  @user_services += names
end

Builder.install do
  # system("systemctl enable --user --now #{@services.join(" ")}")
  # disable all other user services
end

def timer(*names)
  @timers ||= []
  @timers += names
end

Builder.install do
  # timers = @timers.map{ |t| "#{t}.timer" }.join(" ")
  # system("systemctl enable #{timers}")
  # diable all other timers
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
end

Builder.install do
end

# Users and groups commands
def user(name, groups: [])
end

def group(name, sudo: false)
end

# processes commands
def run(command)
end

# files commands
def sync(src, dest)
end

def replace_line(file, pattern, replacement)
end
