require 'set'
require 'etc'

# @group Declarations:

# create a user and assign a set of group. if block is passes the block will run
# in as this user. block will run during the configure step
def user(name, groups: [], autologin: nil, &block)
  name = name.to_s

  @user ||= {}
  @user[name] ||= {}
  @user[name][:groups] ||= []
  @user[name][:groups] += groups.map(&:to_s)
  @user[name][:autologin] = autologin unless autologin.nil?
  @user[name][:state] ||= State.new
  @user[name][:state].apply(block) if block_given?

  on_configure do
    @user.each do |name, conf|
      exists = Etc.getpwnam name rescue nil
      sudo "useradd #{name}" unless exists
      sudo "usermod --groups #{groups.join(",")} #{name}" if groups.any?

      if conf[:autologin]
        FileUtils.mkdir_p '/etc/systemd/system/getty@tty1.service.d'
        file '/etc/systemd/system/getty@tty1.service.d/autologin.conf', <<~FILE
          [Service]
          ExecStart=
          ExecStart=-/sbin/agetty -o '-p -f -- \\u' --noclear --autologin #{name} %I $TERM
        FILE
      end

      fork do
        currentuser = Etc.getpwnam(name)
        Process::GID.change_privilege(currentuser.gid)
        Process::UID.change_privilege(currentuser.uid)
        ENV['XDG_RUNTIME_DIR'] = "/run/user/#{currentuser.uid}"
        ENV['HOME'] = currentuser.dir
        ENV['USER'] = currentuser.name
        ENV['LOGNAME'] = currentuser.name
        conf[:state].run_steps
      end

      Process.wait
    end
  end
end
