require 'set'

# @group Declarations:

# on prepare make sure a git repository is cloned to directory
def git_clone(from:, to: nil)
  @git_clone ||= Set.new
  @git_clone << {from: from, to: to}

  on_install do
    @git_clone.each do |item|
      from = item[:from]
      to = item[:to]
      system "git clone #{from} #{to}" unless File.exist?(File.expand_path(to))
    end
  end
end

# git clone for github repositories
def github_clone(from:, to: nil)
  git_clone(from: "https://github.com/#{from}", to: to)
end
