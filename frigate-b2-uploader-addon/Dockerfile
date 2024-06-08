ARG BUILD_FROM
FROM $BUILD_FROM

# Copy run script
COPY run.sh /run.sh
RUN chmod +x /run.sh

# Set up container environment
RUN mkdir -p /config
COPY init.sh /etc/cont-init.d/init.sh
RUN chmod +x /etc/cont-init.d/init.sh

CMD [ "/run.sh" ]
