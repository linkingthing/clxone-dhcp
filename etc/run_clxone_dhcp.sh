docker run -d --name clxone-dhcp \
--network host \
--restart=always \
-v installpath/etc/clxone-dhcp.conf:/clxone-dhcp.conf \
-v work_key_path:work_key_path \
-v key_factory_path:key_factory_path \
-v /etc/localtime:/etc/localtime \
linkingthing/clxone-dhcp:v2.0.10
