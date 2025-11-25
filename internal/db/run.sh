# Полная пересборка
docker-compose down
docker-compose up --build -d

# Проверьте логи
docker-compose logs -f telegram-bot
