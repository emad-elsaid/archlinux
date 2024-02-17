require 'etc'

# ==============================================================
# UTILITIES:
# functions for logging, tracing, error reporting, coloring text
# ==============================================================

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

def root?
  Process.uid == Etc.getpwnam('root').uid
end

def sudo(command)
  root? ? system(command) : system("sudo #{command}")
end
