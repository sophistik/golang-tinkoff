Цель: Необходимо найти упущенную выгоду

Вход: В файле candles_5m.csv содержится информация об японских свечах (подробнее https://ru.wikipedia.org/wiki/Японские_свечи): тикер - уникальный идентификатор инструмента, время сделки UTC, ohlc, где o - цена открытия на заданном интервале, h - максимальная цена на заданном интервале, l - минимальная цена на заданном интервале, с - цена закрытия на заданном интервале, а в файле users_trades.csv содержится информация о пользовательских сделках (id пользователя, тикер инструмента, цена покупки, цена продажи, время сделки UTC).

Выход: В файле output.csv содержится информация об упущенной выгоде. id - id пользователя из файла users_trades, ticker - тикер инструмента, user_revenue - доход пользователя по данному инструменту, max_revenue - максимально возможный доход по данному инструменту за весь период, diff - упущенная выгода, время, когда надо было продавать бумагу в формате RFC3339, время, когда надо было покупать бумагу в формате RFC3339 (чтобы получить максимальную выгоду).

Детализация: 

     Входящие csv файлы отсортированы по возрастанию по времени сделки
     Порядок записей в выходном файле любой
