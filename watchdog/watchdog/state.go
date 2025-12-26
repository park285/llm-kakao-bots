package watchdog

// SetEnabled updates the global enabled state of the watchdog.
func (w *Watchdog) SetEnabled(enabled bool, requestedBy string, reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cfg.Enabled == enabled {
		return
	}

	w.cfg.Enabled = enabled
	action := "enable"
	if !enabled {
		action = "disable"
	}

	w.logger.Info("watchdog_toggled", "enabled", enabled, "by", requestedBy, "reason", reason)
	w.appendEvent(Event{
		Action:      "watchdog_" + action,
		Container:   "global",
		RequestedBy: requestedBy,
		Reason:      reason,
		Result:      "ok",
	})
}
