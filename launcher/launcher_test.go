package launcher

import (
	"fmt"
	"testing"

	"github.com/homebot/core/urn"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	c := Config{
		Address: "127.0.0.1",
		Secret:  "secret",
		URN:     urn.URN("urn:namespace:service:accountId:resourceType:resource"),
	}

	vars := c.EnvVars()

	assert.Equal(t, map[string]string{
		"SIGMA_HANDLER_ADDRESS": c.Address,
		"SIGMA_ACCESS_SECRET":   c.Secret,
		"SIGMA_INSTANCE_URN":    c.URN.String(),
	}, vars)

	assert.Contains(t, c.Env(), fmt.Sprintf("SIGMA_HANDLER_ADDRESS=%s", c.Address))
	assert.Contains(t, c.Env(), fmt.Sprintf("SIGMA_ACCESS_SECRET=%s", c.Secret))
	assert.Contains(t, c.Env(), fmt.Sprintf("SIGMA_INSTANCE_URN=%s", c.URN.String()))

}
