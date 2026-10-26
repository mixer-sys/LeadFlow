#!/bin/sh
set -e

sed \
  -e "s|\${TELEGRAM_BOT_TOKEN}|${TELEGRAM_BOT_TOKEN}|g" \
  -e "s|\${TELEGRAM_CHAT_ID}|${TELEGRAM_CHAT_ID}|g" \
  /etc/alertmanager/alertmanager.yml.template \
  > /etc/alertmanager/alertmanager.yml

exec /bin/alertmanager \
  --config.file=/etc/alertmanager/alertmanager.yml