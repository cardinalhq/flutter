# Flutter

Flutter takes a script in YAML and generates metrics and other
telemetry to simulate various test scenarios.

A metric emitter is defined that has a start time and various
configuration options (aka "spec") and metrics can refer to
these emitters to form telemetry.  Emitters can be altered
at different points in time.  Metrics can use one or more
emitter to form the datapoint values.
