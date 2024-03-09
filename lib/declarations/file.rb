require 'fileutils'

# @group Declarations:

# Copy src inside dest during configure step, if src/. will copy src content to dest
def copy(src, dest)
  @copy ||= []
  @copy << { src: src, dest: dest }

  on_configure do
    next unless @copy
    next if @copy.empty?

    @copy.each do |item|
      log "Copying", item
      FileUtils.cp_r item[:src], item[:dest]
    end
  end
end

# Replace a regex pattern with replacement string in a file during configure step
def replace(file, pattern, replacement)
  @replace ||= []
  @replace << { file: file, pattern: pattern, replacement: replacement }

  on_configure do
    @replace.each do |params|
      input = File.read(params[:file])
      output = input.gsub(params[:pattern], params[:replacement])
      File.write(params[:file], output)
    end
  end
end

# link file to destination
def symlink(target, link_name)
  @symlink ||= Set.new
  @symlink << { target: target, link_name: link_name }

  on_configure do
    @symlink.each do |params|
      target = File.expand_path params[:target]
      link_name = File.expand_path params[:link_name]

      if File.directory?(target)
        log "Can't link directories", target: target, link_name: link_name
        exit
      end

      log "Linking", target: target, link_name: link_name

      # make the parent if it doesn't exist
      dest_dir = File.dirname(link_name)
      FileUtils.mkdir_p(dest_dir)

      # link with force
      FileUtils.ln_s(target, link_name, force: true)
    end
  end
end

# on prepare make sure the directory exists
def mkdir(*path)
  path.flatten!
  @mkdir ||= Set.new
  @mkdir += path

  on_prepare do
    @mkdir.each do |path|
      FileUtils.mkdir_p File.expand_path(path)
    end
  end
end

# Write a file during configure step
def file(path, content)
  @files ||= {}
  @files[path] = content

  on_configure do
    @files.each do |path, content|
      FileUtils.mkdir_p File.dirname(path)
      File.write(path, content)
    rescue Errno::ENOENT => e
      log "Error: Can't write file", file: path, error: e
    end
  end
end
