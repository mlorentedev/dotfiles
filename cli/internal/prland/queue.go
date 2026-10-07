package prland

import (
	"context"
	"fmt"
)

// LandQueue lands the PRs in the order given, one at a time, and returns one
// Result per PR in that order. A PR that stops, for a refusal or for an error
// reading a fact, is reported and the queue moves on to the next; an error
// becomes the PR's reason. Only a cancelled context ends the queue early, and
// the PRs it never reached say so.
//
// Landing them one at a time is the point. A PR is updated only during its own
// turn, so it pays one update per base move while it is the one landing.
// Concurrent landers each update every PR after every merge: n PRs cost n²
// CI runs and n² reviewer re-reviews.
//
// done, when not nil, is called with each PR's result as soon as it is known,
// so a long queue shows its progress.
func LandQueue(ctx context.Context, o Options, numbers []int, done func(Result)) []Result {
	results := make([]Result, 0, len(numbers))
	for _, n := range numbers {
		var res Result
		if err := ctx.Err(); err != nil {
			res = Result{Number: n, Reasons: []string{fmt.Sprintf("not attempted: %v", err)}}
		} else {
			var err error
			if res, err = Land(ctx, o, n); err != nil {
				res.Merged = false
				res.Reasons = append(res.Reasons, "error: "+err.Error())
			}
		}
		results = append(results, res)
		if done != nil {
			done(res)
		}
	}
	return results
}
