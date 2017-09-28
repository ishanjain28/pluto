FROM debian

RUN apt-get update
RUN apt-get install -y sqlite3  

COPY grafana-4.0.2-1481203731 /
COPY config.ini /

CMD /bin/grafana-server -config=config.ini


 
