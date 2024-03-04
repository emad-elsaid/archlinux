Gem::Specification.new do |s|
  s.authors     = ["Emad Elsaid"]
  s.homepage    = "https://github.com/emad-elsaid/archlinux"
  s.files       = `git ls-files`.lines.map(&:chomp)
  s.name        = 'archlinux'
  s.summary     = "Archlinux DSL to manage whole system state"
  s.version     = '0.0.1'
  s.licenses     = ["GPL-3.0-or-later"]
end
