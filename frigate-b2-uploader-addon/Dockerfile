ARG BUILD_FROM
FROM $BUILD_FROM

# Copy root filesystem
COPY rootfs /

# Execute initialization script
RUN /etc/cont-init.d/init.sh

CMD [ "/run.sh" ]
