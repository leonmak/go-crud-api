package initialize

import (
	"groupbuying.online/api/env"
	"groupbuying.online/api/routes"
)


// TODO: Implement:
// TODO: - rate limit middleware

func Init() {
	env.InitEnv()
	routes.InitRouter()
}
