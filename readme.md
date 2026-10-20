# production-подобный сервис автоматизации заявок


###### cron
'''
0 3 * * * /usr/local/bin/leadflow-backup.sh >> /var/log/leadflow-backup.log 2>&1
'''