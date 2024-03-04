require 'set'

# @group Declarations:

# set timezone and NTP settings during prepare step
def timedate(timezone: 'UTC', ntp: true)
  @timedate = { timezone: timezone, ntp: ntp }

  on_configure do
    log "Set timedate", @timedate
    sudo "timedatectl set-timezone #{@timedate[:timezone]}"
    sudo "timedatectl set-ntp #{@timedate[:ntp]}"
  end
end

# enable system service if root or user service if not during finalize step
def service(*names)
  names.flatten!
  @services ||= Set.new
  @services += names.map(&:to_s)

  on_finalize do
    log "Enable services", services: @services
    user_flags = root? ? "" : "--user"

    system "systemctl enable #{user_flags} #{@services.join(" ")}"

    # Disable services that were enabled manually and not in the list we have
    services = `systemctl list-unit-files #{user_flags} --state=enabled --type=service --no-legend --no-pager`
    enabled_manually = services.lines.map do |l|
                         l.strip.split(/\s+/)
                       end.select { |l| (l[1] == 'enabled') && (l[2] == 'disabled') }
    names_without_extension = enabled_manually.map { |l| l.first.delete_suffix(".service") }
    to_disable = names_without_extension - @services.to_a

    next if to_disable.empty?

    log "Services to disable", packages: to_disable
    # system "systemctl disable #{user_flags} #{to_disable.join(" ")}"
  end
end

# enable system timer if root or user timer if not during finalize step
def timer(*names)
  names.flatten!
  @timers ||= Set.new
  @timers += names.map(&:to_s)

  on_finalize do
    log "Enable timers", timers: @timers
    timers = @timers.map { |t| "#{t}.timer" }.join(" ")
    if root?
      sudo "systemctl enable #{timers}"
    else
      system "systemctl enable --user #{timers}"
    end
    # disable all other timers
  end
end

# set keyboard settings during prepare step
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

  on_prepare do
    next unless @keyboard[:keymap]

    sudo "localectl set-keymap #{@keyboard[:keymap]}"

    m = @keyboard.to_h.slice(:layout, :model, :variant, :options)
    sudo "localectl set-x11-keymap \"#{m[:layout]}\" \"#{m[:model]}\" \"#{m[:variant]}\" \"#{m[:options]}\""
  end
end

# Sets locale using localectl
def locale(value)
  @locale = value

  on_prepare do
    sudo "localectl set-locale #{@locale}"
  end
end

# Sets the machine hostname
def hostname(name)
  @hostname = name

  file '/etc/hostname', "#{@hostname}\n"

  on_configure do
    log "Setting hostname", hostname: @hostname
    sudo "hostnamectl set-hostname #{@hostname}"
  end
end
