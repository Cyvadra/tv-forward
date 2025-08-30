
Write a program to receive alerts from TradingView platform (alert webhook), and send them to different downstreams.


### Support following alert platforms

    - Telegram bot

    - Enterprise Wechat (WeCom) bot

    - Dingtalk bot

    - etc.


### Support following trading platforms

    - Bitget

    - Binance

    - Derbit


### Expected features

    - Use Gin as web framework to receive requests from TradingView

    - Use GORM for database

    - Keep logs for each alert / trading signal

    - Allow forwarding the original message from TradingView to other custum webhook

    - Downstream endpoints are stored in yaml files for user to edit

    - Support trading features like [TVC bot](https://www.tvcbot.com/) platform




