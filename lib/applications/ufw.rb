require 'set'

# @group Declarations:

# setup add ufw enable it and allow ports during configure step
def ufw(*allow)
  @ufw ||= Set.new
  @ufw += allow.map(&:to_s)

  package :ufw
  service :ufw

  on_configure do
    @ufw.each do |port|
      sudo "ufw allow #{port}"
    end
  end
end
