require 'etc'

# @group Utilities

# Prints a message to the STDOUT
# @param msg [String] a log message to print
# @param args [Hash<String, Object>] prints each key and value in separate lines after message
def log(msg, args={})
  puts msg

  return unless args.any?

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

# Checks if current user is the root
# @return [Boolean] true if current user is root and false otherwise
def root?
  Process.uid == Etc.getpwnam('root').uid
end

# Runs the command with sudo if current user is not root
def sudo(command)
  root? ? system(command) : system("sudo #{command}")
end
