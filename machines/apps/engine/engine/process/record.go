package process

import "github.com/jterrazz/studio-station/apps/engine/engine"

// invariant: a record that cannot be written never stops the engine: the
// window then finds station by the install folder alone, and the log says why.
func (d *Engine) keepRecord() {
	if d.o.Records == nil {
		return
	}
	r := engine.Record{V: engine.RecordVersion, Executable: d.o.Executable, Version: d.o.Version, At: d.started}
	if err := d.o.Records.Write(r); err != nil {
		d.logf("record", "could not keep the binary it started from: "+err.Error())
	}
}
