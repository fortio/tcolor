FROM scratch
COPY tcolor /usr/bin/tcolor
ENTRYPOINT ["/usr/bin/tcolor"]
