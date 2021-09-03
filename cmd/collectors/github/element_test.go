package github

import (
    "testing"
)

func TestParseElement(t *testing.T) {

    var (
        in string
        e *Element
    )

    in = "^name   =>    display "
    e = ParseElement(in)

    if e.Name == "name" && e.DisplayName == "display" && e.IsLabel && ! e.IsKey && !e.IsMetric {
        t.Logf("OK [%s]: [%s]", in, e.String())
    } else {
        t.Errorf("FAIL [%s]: [%s]", in, e.String())
    }

    in = "=> repo"
    e = ParseElement(in)
    if e.Name == "" && e.DisplayName == "repo" && !e.IsLabel && !e.IsKey && e.IsMetric {
        t.Logf("OK [%s]: [%s]", in, e.String())
    } else {
        t.Errorf("FAIL [%s]: [%s]", in, e.String())
    }

    in = "^^referrer"
    e = ParseElement(in)
    if e.Name == "referrer" && e.DisplayName == "referrer" && e.IsLabel && e.IsKey && !e.IsMetric {
        t.Logf("OK [%s]: [%s]", in, e.String())
    } else {
        t.Errorf("FAIL [%s]: [%s]", in, e.String())
    }
}
