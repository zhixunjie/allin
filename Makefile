.PHONY: up down down-v build logs logs-server logs-web ps dev-server dev-ui

up: ## 一键启动所有服务（首次运行会自动构建镜像）
	docker compose up -d

down: ## 停止并移除容器（数据库数据保留）
	docker compose down

down-v: ## 停止并移除容器及数据卷（清空数据库）
	docker compose down -v

build: ## 重新构建所有镜像（代码更新后执行）
	docker compose build

logs: ## 实时查看所有服务日志
	docker compose logs -f

logs-server: ## 实时查看后端日志
	docker compose logs -f server

logs-web: ## 实时查看 nginx 日志
	docker compose logs -f web

ps: ## 查看各服务运行状态
	docker compose ps

dev-server: ## 本地开发：启动后端（依赖本地 MySQL）
	cd server && go run ./base/

dev-ui: ## 本地开发：启动前端（Vite dev server）
	cd ui-web && npm run dev
