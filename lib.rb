# State of the system
# It should hold all the information we need to build the system, packages, files, changes...etc
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
      step.call state
    end
  end
end

def linux(&block)
  s = State.new
  s.apply(block)
  Builder.new(s).run
end


# package command, accumulates packages needs to be installed
class State
  attr_accessor :packages

  def package(*names)
    self.packages ||= []
    self.packages += names
  end
end

# install step to install packages required and remove not required
Builder.install do |state|
  # install packages list as is
  names = state.packages.join(" ")
  system("sudo pacman --needed -S #{names}") unless state.packages.empty?

  # expand groups to packages
  group_packages = `pacman --quiet -Sg #{names}`.lines.map(&:strip)

  # full list of packages that should exist on the system
  all = (state.packages + group_packages).uniq

  # actual list on the system
  # TODO this list doesn't include packages that were
  # explicitly installed AND is a dependency at the same time like `linux` which is dependency of `base`
  # or `gimp` which is optional dependency of `alsa-lib`
  installed = `pacman -Q --quiet --explicit --unrequired --native`.lines.map(&:strip)

  unneeded = installed - all
  `sudo pacman -Rs #{unneeded.join(" ")}` unless unneeded.empty?
end

# aur command to install packages from aur
class State
  attr_accessor :aurs

  def aur(*names)
    self.aurs ||= ['yay']
    self.aurs += names
  end
end

Builder.install do |state|
  names = state.aurs
  flags = '--norebuild --noredownload --editmenu=false --diffmenu=false --noconfirm'
  system("yay #{flags} -S #{names.join(" ")}") unless names.empty?

  installed = `pacman -Q --quiet --explicit --unrequired --foreign`.lines.map(&:strip)
  unneeded = installed - names
  `sudo pacman -Rs #{unneeded.join(" ")}` unless unneeded.empty?
end

# Systemd
class State
  def timezone(tz)
    @timezone = tz
  end

  def service(*names)
    @services ||= []
    @services += names
  end
end

Builder.install do |state|
end
