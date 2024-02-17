# ==============================================================
# CORE:
# State of the system It should hold all the information we need to build the
# system, packages, files, changes...etc. everything will run inside an instance
# of this class
# ==============================================================
class State
  def apply(block)
    instance_eval &block
  end

  # Run block on prepare step. id identifies the block uniqueness in the steps.
  # registering a block with same id multiple times replaces old block by new
  # one. if id is nil the block location in source code is used as an id
  def on_prepare(id=nil, &block)
    id ||=  caller_locations(1,1).first.to_s
    @prepare_steps ||= {}
    @prepare_steps[id] = block
  end

  # Same as on_prepare but for install step
  def on_install(id=nil, &block)
    id ||=  caller_locations(1,1).first.to_s
    @install_steps ||= {}
    @install_steps[id] = block
  end

  # Same as on_prepare but for configure step
  def on_configure(id=nil, &block)
    id ||=  caller_locations(1,1).first.to_s
    @configure_steps ||= {}
    @configure_steps[id] = block
  end

  # Same as on_finalize but for configure step
  def on_finalize(id=nil, &block)
    id ||=  caller_locations(1,1).first.to_s
    @finalize_steps ||= {}
    @finalize_steps[id] = block
  end

  # Run all registered code blocks in the following order: Prepare, Install, Configure, Finalize
  def run_steps
    if @prepare_steps&.any?
      log "=> Prepare"
      @prepare_steps.each { |_, step| apply(step) }
    end

    if @install_steps&.any?
      log "=> Install"
      @install_steps.each { |_, step| apply(step) }
    end

    if @configure_steps&.any?
      log "=> Configure"
      @configure_steps.each { |_, step| apply(step) }
    end

    if @finalize_steps&.any?
      log "=> Finalize"
      @finalize_steps.each { |_, step| apply(step) }
    end
  end
end

# passed block will run in the context of a State instance and then a builder
# will build this state
def linux(&block)
  s = State.new
  s.apply(block)
  s.run_steps
end