-- データベースにUserテーブル作成
CREATE TABLE Users ( 
	id INT64 NOT NULL,
	name STRING(16) NOT NULL,
	money FLOAT64
) PRIMARY KEY (id);

-- データ格納
INSERT INTO `Users`(`id`, `name`, `money`) VALUES(1, 'takahira', 10000);
INSERT INTO `Users`(`id`, `name`, `money`) VALUES(2, 'test-user-1', 1000);
INSERT INTO `Users`(`id`, `name`, `money`) VALUES(3, 'test-user-2', 2000);
INSERT INTO `Users`(`id`, `name`, `money`) VALUES(4, 'test-user-3', 3000);

-- Userテーブルデータ一覧表示
SELECT * FROM `Users` ORDER BY id ASC;