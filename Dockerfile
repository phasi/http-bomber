FROM scratch

COPY ./dist/http-bomber_linux_amd64 /usr/local/bin/http-bomber

ENTRYPOINT [ "/usr/local/bin/http-bomber" ]
