CREATE TABLE IF NOT EXISTS  `log_position` (
  `id` varchar(255) NOT NULL,
  `operator` int(11) NOT NULL,
  `server` int(11) NOT NULL,
  `log` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `last_execute` date DEFAULT NULL,
  `position` bigint(20) NOT NULL,
  `total_rows` int(11) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;