-- MySQL dump 10.13  Distrib 8.0.20, for macos10.15 (x86_64)
--
-- Host: localhost    Database: register/**/
-- ------------------------------------------------------
-- Server version	8.0.21

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `register`
--

/*!40000 DROP DATABASE IF EXISTS `register`*/;

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `register` /*!40100 DEFAULT CHARACTER SET utf8 */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `register`;

--
-- Table structure for table `columns`
--

DROP TABLE IF EXISTS `columns`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `columns` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(100) DEFAULT NULL,
  `color` varchar(45) NOT NULL,
  `column_index` int NOT NULL,
  `letter` varchar(45) NOT NULL,
  `is_category` tinyint NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id_UNIQUE` (`id`),
  UNIQUE KEY `column_id_UNIQUE` (`column_index`),
  UNIQUE KEY `name_UNIQUE` (`name`),
  KEY `idx_columns_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=56 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `columns`
--

LOCK TABLES `columns` WRITE;
/*!40000 ALTER TABLE `columns` DISABLE KEYS */;
INSERT INTO `columns` (`id`, `name`, `color`, `column_index`, `letter`, `is_category`, `created_at`, `updated_at`, `deleted_at`)
VALUES (1,'Reconciled','white',0,'A',0,'2020-11-24 15:11:30.139','2020-11-24 15:11:30.139',NULL),
(2,'Check','white',1,'B',0,'2020-11-24 15:11:30.140','2020-11-24 15:11:30.140',NULL),
(3,'Date','white',2,'C',0,'2020-11-24 15:11:30.142','2020-11-24 15:11:30.142',NULL),
(4,'Description','white',3,'D',0,'2020-11-24 15:11:30.143','2020-11-24 15:11:30.143',NULL),
(5,'Withdrawals','white',4,'E',0,'2020-11-24 15:11:30.144','2020-11-24 15:11:30.144',NULL),
(6,'Deposits','white',5,'F',0,'2020-11-24 15:11:30.144','2020-11-24 15:11:30.144',NULL),
(7,'Credit Purchases','white',6,'G',0,'2020-11-24 15:11:30.145','2020-11-24 15:11:30.145',NULL),
(8,'Register','white',7,'H',0,'2020-11-24 15:11:30.146','2020-11-24 15:11:30.146',NULL),
(9,'Cleared','white',8,'I',0,'2020-11-24 15:11:30.147','2020-11-24 15:11:30.147',NULL),
(10,'Delta','white',9,'J',0,'2020-11-24 15:11:30.148','2020-11-24 15:11:30.148',NULL),
(11,'Cash','green',10,'K',1,'2020-11-24 15:11:30.149','2020-11-24 15:11:30.149',NULL),
(12,'Dining Out','green',11,'L',1,'2020-11-24 15:11:30.149','2020-11-24 15:11:30.149',NULL),
(13,'Gas','green',12,'M',1,'2020-11-24 15:11:30.150','2020-11-24 15:11:30.150',NULL),
(14,'Grocery','green',13,'N',1,'2020-11-24 15:11:30.151','2020-11-24 15:11:30.151',NULL),
(15,'Misc','green',14,'O',1,'2020-11-24 15:11:30.152','2020-11-24 15:11:30.152',NULL),
(16,'Vape Supplies','green',15,'P',1,'2020-11-24 15:11:30.152','2020-11-24 15:11:30.152',NULL),
(17,'AT&T Cell Phone','yellow',16,'Q',1,'2020-11-24 15:11:30.153','2020-11-24 15:11:30.153',NULL),
(18,'Content Subscriptions','yellow',17,'R',1,'2020-11-24 15:11:30.154','2020-11-24 15:11:30.154',NULL),
(19,'Comcast/Xfinity Internet','yellow',18,'S',1,'2020-11-24 15:11:30.154','2020-11-24 15:11:30.154',NULL),
(20,'Dad DSL (old)','yellow',19,'T',1,'2020-11-24 15:11:30.155','2020-11-24 15:11:30.155',NULL),
(21,'Washington Gas','yellow',20,'U',1,'2020-11-24 15:11:30.156','2020-11-24 15:11:30.156',NULL),
(22,'Dominion Power','yellow',21,'V',1,'2020-11-24 15:11:30.156','2020-11-24 15:11:30.156',NULL),
(23,'Hair Cut','yellow',22,'W',1,'2020-11-24 15:11:30.157','2020-11-24 15:11:30.157',NULL),
(24,'Harley Davidson (old)','yellow',23,'X',1,'2020-11-24 15:11:30.158','2020-11-24 15:11:30.158',NULL),
(25,'Car Insurance','yellow',24,'Y',1,'2020-11-24 15:11:30.158','2020-11-24 15:11:30.158',NULL),
(26,'Motorcycle Insurance (old)','yellow',25,'Z',1,'2020-11-24 15:11:30.159','2020-11-24 15:11:30.159',NULL),
(27,'Massage','yellow',26,'AA',1,'2020-11-24 15:11:30.160','2020-11-24 15:11:30.160',NULL),
(28,'Loudoun Heights Rent','yellow',27,'AB',1,'2020-11-24 15:11:30.161','2020-11-24 15:11:30.161',NULL),
(29,'Renters Insurance','yellow',28,'AC',1,'2020-11-24 15:11:30.161','2020-11-24 15:11:30.161',NULL),
(30,'Storage Rental','yellow',29,'AD',1,'2020-11-24 15:11:30.162','2020-11-24 15:11:30.162',NULL),
(31,'Credit Cards','yellow',30,'AE',1,'2020-11-24 15:11:30.163','2020-11-24 15:11:30.163',NULL),
(32,'Citi Bank (old)','yellow',31,'AF',1,'2020-11-24 15:11:30.164','2020-11-24 15:11:30.164',NULL),
(33,'Personal Loan','yellow',32,'AG',1,'2020-11-24 15:11:30.165','2020-11-24 15:11:30.165',NULL),
(34,'Car Loan','yellow',33,'AH',1,'2020-11-24 15:11:30.165','2020-11-24 15:11:30.165',NULL),
(35,'IRS','yellow',34,'AI',1,'2020-11-24 15:11:30.166','2020-11-24 15:11:30.166',NULL),
(36,'Smart Tag','yellow',35,'AJ',1,'2020-11-24 15:11:30.167','2020-11-24 15:11:30.167',NULL),
(37,'Supplements (old)','yellow',36,'AK',1,'2020-11-24 15:11:30.168','2020-11-24 15:11:30.168',NULL),
(38,'Car Expenses','blue',37,'AL',1,'2020-11-24 15:11:30.169','2020-11-24 15:11:30.169',NULL),
(39,'Car Property Tax','blue',38,'AM',1,'2020-11-24 15:11:30.169','2020-11-24 15:11:30.169',NULL),
(40,'Clothing & Household','blue',39,'AN',1,'2020-11-24 15:11:30.170','2020-11-24 15:11:30.170',NULL),
(41,'Extra','blue',40,'AO',1,'2020-11-24 15:11:30.171','2020-11-24 15:11:30.171',NULL),
(42,'Gifts','blue',41,'AP',1,'2020-11-24 15:11:30.171','2020-11-24 15:11:30.171',NULL),
(43,'Meg (old)','blue',42,'AQ',1,'2020-11-24 15:11:30.172','2020-11-24 15:11:30.172',NULL),
(44,'Nickolas (old)','blue',43,'AR',1,'2020-11-24 15:11:30.173','2020-11-24 15:11:30.173',NULL),
(45,'Maria (old)','blue',44,'AS',1,'2020-11-24 15:11:30.174','2020-11-24 15:11:30.174',NULL),
(46,'Gandalf','blue',45,'AT',1,'2020-11-24 15:11:30.175','2020-11-24 15:11:30.175',NULL),
(47,'Mental Health','blue',46,'AU',1,'2020-11-24 15:11:30.177','2020-11-24 15:11:30.177',NULL),
(48,'Medical (SoberLink)','blue',47,'AV',1,'2020-11-24 15:11:30.177','2020-11-24 15:11:30.177',NULL),
(49,'Vision','blue',48,'AW',1,'2020-11-24 15:11:30.178','2020-11-24 15:11:30.178',NULL),
(50,'Prescriptions (old)','blue',49,'AX',1,'2020-11-24 15:11:30.179','2020-11-24 15:11:30.179',NULL),
(51,'Emergency Fund','blue',50,'AY',1,'2020-11-24 15:11:30.179','2020-11-24 15:11:30.179',NULL),
(52,'Vacation','blue',51,'AZ',1,'2020-11-24 15:11:30.180','2020-11-24 15:11:30.180',NULL),
(53,'Exercise Equipment','blue',52,'BA',1,'2020-11-24 15:11:30.181','2020-11-24 15:11:30.181',NULL),
(54,'General Savings (Court Fines)','blue',53,'BB',1,'2020-11-24 15:11:30.182','2020-11-24 15:11:30.182',NULL);
/*!40000 ALTER TABLE `columns` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `merchants`
--

DROP TABLE IF EXISTS `merchants`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `merchants` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `bank_name` varchar(100) NOT NULL,
  `column_id` int NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id_UNIQUE` (`id`),
  KEY `bank_name_UNIQUE` (`bank_name`),
  KEY `column_id_fk_idx` (`column_id`),
  KEY `idx_merchants_deleted_at` (`deleted_at`),
  CONSTRAINT `column_id_fk` FOREIGN KEY (`column_id`) REFERENCES `columns` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=64 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `merchants`
--

LOCK TABLES `merchants` WRITE;
/*!40000 ALTER TABLE `merchants` DISABLE KEYS */;
INSERT INTO `merchants` (`id`, `name`, `bank_name`, `column_id`, `created_at`, `updated_at`, `deleted_at`) VALUES (1,'Amazon','AMAZON.COM',40,'2020-11-24 15:11:30.217','2020-11-24 15:11:30.217',NULL),
(2,'Amazon','Amazon.com',40,'2020-11-24 15:11:30.218','2020-11-24 15:11:30.218',NULL),
(3,'Amazon','AMZN Mktp',40,'2020-11-24 15:11:30.220','2020-11-24 15:11:30.220',NULL),
(4,'Amazon Web Services','Amazon web services',15,'2020-11-24 15:11:30.222','2020-11-24 15:11:30.222',NULL),
(5,'Apple iTunes','APPLE.COM/BILL',15,'2020-11-24 15:11:30.223','2020-11-24 15:11:30.223',NULL),
(6,'AT&T iPhone','BILL PAY AT&T',17,'2020-11-24 15:11:30.225','2020-11-24 15:11:30.225',NULL),
(7,'Comcast/Xfinity','BILL PAY COMCAST',19,'2020-11-24 15:11:30.226','2020-11-24 15:11:30.226',NULL),
(8,'Washington Gas','BILL PAY WASHINGTON GAS',21,'2020-11-24 15:11:30.227','2020-11-24 15:11:30.227',NULL),
(9,'Blue Mount Nursery','BLUE MOUNT NURSERY',40,'2020-11-24 15:11:30.228','2020-11-24 15:11:30.228',NULL),
(10,'Cascades Center for Dentistry','CASCADES CNTR FOR DENT',41,'2020-11-24 15:11:30.229','2020-11-24 15:11:30.229',NULL),
(11,'Chewy','CHEWY',14,'2020-11-24 15:11:30.230','2020-11-24 15:11:30.230',NULL),
(12,'City of Vape','CITY OF VAPE',16,'2020-11-24 15:11:30.231','2020-11-24 15:11:30.231',NULL),
(13,'Costco','COSTCO WHSE',14,'2020-11-24 15:11:30.232','2020-11-24 15:11:30.232',NULL),
(14,'CubeSmart','CUBESMART',30,'2020-11-24 15:11:30.233','2020-11-24 15:11:30.233',NULL),
(15,'Dominion Power','DOMINION POWER',22,'2020-11-24 15:11:30.235','2020-11-24 15:11:30.235',NULL),
(16,'Exxon Gas','EXXONMOBIL',21,'2020-11-24 15:11:30.236','2020-11-24 15:11:30.236',NULL),
(17,'EZ Pass','E Z PASS',36,'2020-11-24 15:11:30.238','2020-11-24 15:11:30.238',NULL),
(18,'Fidelity Visa','Fidelity Visa',31,'2020-11-24 15:11:30.239','2020-11-24 15:11:30.239',NULL),
(19,'Fracture','FRACTURE',40,'2020-11-24 15:11:30.240','2020-11-24 15:11:30.240',NULL),
(20,'Harmony Hill Farms','HARMONY HILL',14,'2020-11-24 15:11:30.241','2020-11-24 15:11:30.241',NULL),
(21,'Harris Teeter','HARRIS TEETER',14,'2020-11-24 15:11:30.242','2020-11-24 15:11:30.242',NULL),
(22,'Homesite Renters Insurance','HOMESITE INS PREM',29,'2020-11-24 15:11:30.243','2020-11-24 15:11:30.243',NULL),
(23,'Hulu','HULU',18,'2020-11-24 15:11:30.245','2020-11-24 15:11:30.245',NULL),
(24,'HP Instant Ink','INSTANT INK',15,'2020-11-24 15:11:30.246','2020-11-24 15:11:30.246',NULL),
(25,'IRS Tax Payment','IRS USATAXPYMT',35,'2020-11-24 15:11:30.247','2020-11-24 15:11:30.247',NULL),
(26,'Chase Subaru Car Payment','JPMorgan Chase Ext Trnsfr',34,'2020-11-24 15:11:30.248','2020-11-24 15:11:30.248',NULL),
(27,'Light In The Box','LIGHTINTHEBOX',40,'2020-11-24 15:11:30.250','2020-11-24 15:11:30.250',NULL),
(28,'Loudoun County District Court','LOUDOUN COUNTY GENERAL',54,'2020-11-24 15:11:30.251','2020-11-24 15:11:30.251',NULL),
(29,'Loudoun Club 12 - PayPal','LOUDOUNCLUB',54,'2020-11-24 15:11:30.253','2020-11-24 15:11:30.253',NULL),
(30,'Loudoun Heights Rent','Loudoun Heights',28,'2020-11-24 15:11:30.254','2020-11-24 15:11:30.254',NULL),
(31,'Deposit Check','MOBILE DEPOSIT',51,'2020-11-24 15:11:30.255','2020-11-24 15:11:30.255',NULL),
(32,'Netflix','NETFLIX',18,'2020-11-24 15:11:30.257','2020-11-24 15:11:30.257',NULL),
(33,'Plex Pass','PLEXINCPASS',18,'2020-11-24 15:11:30.260','2020-11-24 15:11:30.260',NULL),
(34,'Progressive Auto Insurance','PROG ADVANCED INS PREM',25,'2020-11-24 15:11:30.261','2020-11-24 15:11:30.261',NULL),
(35,'RedBox','REDBOX',15,'2020-11-24 15:11:30.262','2020-11-24 15:11:30.262',NULL),
(36,'SlingTV','SLING.COM',18,'2020-11-24 15:11:30.264','2020-11-24 15:11:30.264',NULL),
(37,'SnapSure Storage Rental Ins','SNAPNSURE INSURANCE',30,'2020-11-24 15:11:30.265','2020-11-24 15:11:30.265',NULL),
(38,'SoFi Personal Loan','SOFI PAYMENTS',33,'2020-11-24 15:11:30.266','2020-11-24 15:11:30.266',NULL),
(39,'Spotify','Spotify',18,'2020-11-24 15:11:30.267','2020-11-24 15:11:30.267',NULL),
(40,'The Fermented Pig','THE FERMENTED PIG',14,'2020-11-24 15:11:30.269','2020-11-24 15:11:30.269',NULL),
(41,'The UPS Store','THE UPS STORE',15,'2020-11-24 15:11:30.270','2020-11-24 15:11:30.270',NULL),
(42,'Valencia - Farmers Market','VALENCIA',14,'2020-11-24 15:11:30.271','2020-11-24 15:11:30.271',NULL),
(43,'Venmo Payment','VENMO PAYMENT',54,'2020-11-24 15:11:30.273','2020-11-24 15:11:30.273',NULL),
(44,'VueMastery.com','VUEMASTERY',41,'2020-11-24 15:11:30.275','2020-11-24 15:11:30.275',NULL),
(45,'Walmart.com','WALMART.COM',40,'2020-11-24 15:11:30.276','2020-11-24 15:11:30.276',NULL),
(46,'Cash','WITHDRAWAL MADE IN A BRANCH/STORE',11,'2020-11-24 15:11:30.278','2020-11-24 15:11:30.278',NULL),
(47,'Sierra Club','SIERRA CLUB',42,NULL,NULL,NULL),
(50,'Amazon','Amazon',40,'2020-11-25 13:13:28.888','2020-11-25 13:13:28.888',NULL),
(51,'Calzedonia','FRGN TRANS FEE-CALZEDONIA SPA',40,'2020-11-25 13:13:43.249','2020-11-25 13:13:43.249',NULL),
(52,'Calzedonia','CALZEDONIA SPA MALCESINE',40,'2020-11-25 13:13:56.423','2020-11-25 13:13:56.423',NULL),
(53,'Harris Teeter','Harris Teeter Supermarkets, Inc.',14,'2020-11-25 13:14:03.911','2020-11-25 13:14:03.911',NULL),
(56,'Loudoun Club 12\n','LOUDOUN CLUB 1',42,'2020-11-28 11:26:28.649','2020-11-28 11:26:28.649',NULL),
(57,'Costco\n','Costco',14,'2020-11-28 11:26:40.106','2020-11-28 11:26:40.106',NULL),
(60,'CrowdStrike Salary','CROWDSTRIKE INC',11,NULL,NULL,NULL),
(61,'Vape Juice','VJUICETOBACCOPRODUCT',11,NULL,NULL,NULL),
(62,'Chase Visa','BILL PAY CHASE CARD SERVICES ON-LINE xxxxxxxxxxx35652 ON 12-01',31,NULL,NULL,NULL),
(63,'Credit Card Payment (delete)','Payment Thank You - Bill',11,NULL,NULL,NULL);
/*!40000 ALTER TABLE `merchants` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2020-12-03 13:44:07
