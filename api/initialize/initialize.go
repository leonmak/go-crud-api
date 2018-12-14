package initialize

import (
	"groupbuying.online/api/env"
	"groupbuying.online/api/routes"
)


// TODO: Implement:
// TODO: - csrf, rate limit middleware
// TODO: - cloudinary, email verification, change name/password

func Init() {
	env.InitEnv()
	routes.InitRouter()
}
