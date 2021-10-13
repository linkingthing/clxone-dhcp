docker run -d --name clxone-dhcp \
--network host \
--restart=always \
-p 58085:58085 \
-p 58885:58885 \
-v installpath/etc/clxone-dhcp.conf:/clxone-dhcp.conf \
-v /etc/localtime:/etc/localtime \
linkingthing/clxone-dhcp:v1.6.0
