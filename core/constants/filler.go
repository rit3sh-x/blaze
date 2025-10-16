package constants

import (
	"fmt"
)

var EnvContent = fmt.Sprintf(`# Your Database URL
%s=postgresql://user:password@localhost:5432/mydb

# Connection configs
# %s=0
# %s=25
`, DATABASE_URI_ENV, DB_MIN_CONNS_ENV, DB_MAX_CONNS_ENV)