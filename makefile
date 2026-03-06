dc-up:
	docker-compose up -d

dc-up-b:
	docker-compose up -d --build

dc-down:
	docker-compose down

migrations:
	cat migrations/0001_init_schema.sql | docker exec -i telegram_bot_senya-postgres-1 psql -U postgres -d telegram_bot

ds_pg:
	docker exec -it telegram_bot_senya-postgres-1 psql -U postgres -d telegram_bot