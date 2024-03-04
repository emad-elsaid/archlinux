def require_relative_dir(dir)
  Dir["#{File.dirname(__FILE__)}/#{dir}/**/*.rb"].each { |f| require f }
end

require_relative 'core'
require_relative 'utils'
require_relative_dir 'declarations'
require_relative_dir 'applications'

Signal.trap("INT") { exit } # Suppress stack trace on Ctrl-C
