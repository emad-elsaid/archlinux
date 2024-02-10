require_relative 'core'
require_relative 'utils'
require_relative 'declarations'

Signal.trap("INT") { exit } # Suppress stack trace on Ctrl-C
