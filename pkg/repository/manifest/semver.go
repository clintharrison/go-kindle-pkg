package manifest

import (
	"encoding/json"
	"fmt"

	"github.com/pingcap/errors"
)

type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

func (sv *SemanticVersion) Compare(other SemanticVersion) int {
	if sv.Major != other.Major {
		return sv.Major - other.Major
	}
	if sv.Minor != other.Minor {
		return sv.Minor - other.Minor
	}
	return sv.Patch - other.Patch
}

func (sv *SemanticVersion) UnmarshalJSON(data []byte) error {
	var vs []int
	err := json.Unmarshal(data, &vs)
	if err != nil {
		return errors.AddStack(err)
	}
	if len(vs) != 3 {
		return errors.New("semver expected to be [major, minor, patch]")
	}
	sv.Major = vs[0]
	sv.Minor = vs[1]
	sv.Patch = vs[2]
	return nil
}

func (sv *SemanticVersion) MarshalJSON() ([]byte, error) {
	vs := []int{sv.Major, sv.Minor, sv.Patch}
	return json.Marshal(vs)
}

func (sv *SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
}
