#!/bin/bash
set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Загружаем конфигурацию
if [ ! -f "deploy.conf" ]; then
    echo -e "${RED}Ошибка: deploy.conf не найден${NC}"
    echo "Скопируй deploy.conf.example в deploy.conf и заполни данные"
    exit 1
fi

source deploy.conf

# Проверяем обязательные переменные
if [ -z "$DEPLOY_HOST" ] || [ -z "$DEPLOY_USER" ] || [ -z "$DEPLOY_PATH" ]; then
    echo -e "${RED}Ошибка: Не заполнены обязательные переменные в deploy.conf${NC}"
    exit 1
fi

SSH_CMD="ssh -p ${DEPLOY_PORT:-22} ${DEPLOY_USER}@${DEPLOY_HOST}"

echo -e "${YELLOW}=== Деплой Weather на ${DEPLOY_HOST} ===${NC}"

# Проверяем SSH подключение
echo -e "${GREEN}[1/5] Проверяю SSH подключение...${NC}"
$SSH_CMD "echo 'SSH OK'" || {
    echo -e "${RED}Не удалось подключиться по SSH${NC}"
    exit 1
}

# Проверяем есть ли репозиторий на сервере
echo -e "${GREEN}[2/5] Проверяю репозиторий на сервере...${NC}"
$SSH_CMD "
    if [ ! -d ${DEPLOY_PATH} ]; then
        echo 'Клонирую репозиторий...'
        git clone ${GIT_REPO:-https://github.com/iRootPro/weather.git} ${DEPLOY_PATH}
    fi
"

# Обновляем код
echo -e "${GREEN}[3/5] Обновляю код из ${GIT_BRANCH:-main}...${NC}"
$SSH_CMD "
    cd ${DEPLOY_PATH}
    git fetch origin
    git checkout ${GIT_BRANCH:-main}
    git pull origin ${GIT_BRANCH:-main}
"

# Проверяем .env
echo -e "${GREEN}[4/5] Проверяю .env...${NC}"
$SSH_CMD "
    cd ${DEPLOY_PATH}
    if [ ! -f .env ]; then
        echo -e '${RED}ВНИМАНИЕ: .env не найден!${NC}'
        echo 'Создай .env на сервере: cp .env.example .env && nano .env'
        exit 1
    fi
"

# Пересобираем и перезапускаем контейнеры
echo -e "${GREEN}[5/5] Пересобираю и перезапускаю контейнеры...${NC}"
$SSH_CMD "
    cd ${DEPLOY_PATH}
    docker-compose -f docker-compose.prod.yml build
    docker-compose -f docker-compose.prod.yml up -d

    echo ''
    echo '=== Статус контейнеров ==='
    docker-compose -f docker-compose.prod.yml ps
"

echo ""
echo -e "${GREEN}=== Деплой завершён успешно! ===${NC}"
echo -e "Логи: ${YELLOW}make deploy-logs${NC}"
