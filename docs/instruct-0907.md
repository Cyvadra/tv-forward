Alert signals (without trading) are in json format with a certain "message" field, which is the only field that will be used in program. You can judge whether an alert triggers trading by validating required fields for trading.

Trading signals are sent by TradingView in json format like `docs/tvcbot.json`. Modify program or even refactor structures, procedures and procedures to adapt it.

Use `api_sec` for identification. Each `api_sec` stands for an isolated user account, whose trades and signals can be shown and summarized separately. In other words, each `api_sec` has its corresponding "trading platform", "api keys", leverage and other data. Alert parts can be remained the same without authorization.

You can see from the given `tvcbot.json`, that it has some tricks here to trace position changes and avoid mistakes like repeated trades or network error. So when you open a position, check "prev_market_position_size" first. The final goal is to set account position to the latest position size. Fields in `tvcbot.json` are mostly not redundant, analyze them and add features if possible.

Initialize another yaml to store users ( with each `api_sec`, account credentials, etc. ). Preserve received signals and operated trading records (even failed ones).

Platforms including Binance, Bitget, OKX shall be supported.

Since you're gonna work on this project, optimize and simplify codes by the way.