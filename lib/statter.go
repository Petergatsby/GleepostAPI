package lib

import (
	"time"

	"github.com/peterbourgon/g2s"
)

//PrefixStatter can report things to statsd with a prefix on their bucket name
type PrefixStatter struct {
	statter         g2s.Statter
	DevelopmentMode bool
}

func (statter PrefixStatter) prefix() string {
	if statter.DevelopmentMode {
		return "dev."
	}
	return "prod."
}

//Time reports the time for this stat to statsd. (use it with defer)
func (statter PrefixStatter) Time(start time.Time, bucket string) {
	//TODO: Move the stats stuff into its own module?
	duration := time.Since(start)
	bucket = statter.prefix() + bucket
	if statter.statter != nil {
		statter.statter.Timing(1.0, bucket, duration)
	}
}

//Count wraps a g2s.Statter giving an automatic version prefix and a single location to set the report probability.
func (statter PrefixStatter) Count(count int, bucket string) {
	if statter.statter != nil {
		statter.statter.Counter(1.0, statter.prefix()+bucket, count)
	}
}

//Gauge sends a statsd gauge with a fixed probability and prefix.
func (statter PrefixStatter) Gauge(gauge string, bucket string) {
	if statter.statter != nil {
		statter.statter.Gauge(1.0, statter.prefix()+bucket, gauge)
	}
}
