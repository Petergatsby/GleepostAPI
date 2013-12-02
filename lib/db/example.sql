-- phpMyAdmin SQL Dump
-- version 3.3.7deb7
-- http://www.phpmyadmin.net
--
-- Host: localhost
-- Generation Time: Oct 10, 2013 at 04:37 PM
-- Server version: 5.1.63
-- PHP Version: 5.3.3-7+squeeze14

SET SQL_MODE="NO_AUTO_VALUE_ON_ZERO";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;

--
-- Database: `gleepost`
--

-- --------------------------------------------------------

--
-- Table structure for table `chat_messages`
--

CREATE TABLE IF NOT EXISTS `chat_messages` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `conversation_id` int(10) unsigned NOT NULL,
  `from` int(10) unsigned NOT NULL,
  `text` varchar(1024) COLLATE utf8_bin NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `seen` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=87 ;

--
-- Dumping data for table `chat_messages`
--

INSERT INTO `chat_messages` (`id`, `conversation_id`, `from`, `text`, `timestamp`, `seen`) VALUES
(4, 5, 9, 'testing', '2013-09-09 18:16:57', 0),
(5, 5, 9, 'testing', '2013-09-09 18:18:12', 0),
(6, 5, 9, 'testing 123', '2013-09-10 13:35:59', 0),
(7, 5, 9, 'weeeee', '2013-09-11 17:49:01', 0),
(8, 5, 9, 'weeeee', '2013-09-11 17:49:01', 0),
(9, 5, 9, 'weeeee', '2013-09-11 17:49:02', 0),
(10, 5, 9, 'weeeee', '2013-09-11 17:49:02', 0),
(11, 5, 9, 'weeeee', '2013-09-11 17:49:02', 0),
(12, 5, 9, 'weeeee', '2013-09-11 17:49:02', 0),
(13, 5, 9, 'weeeee', '2013-09-11 17:49:02', 0),
(14, 5, 9, 'weeeee', '2013-09-11 17:49:03', 0),
(15, 5, 9, 'weeeee', '2013-09-11 17:49:03', 0),
(16, 5, 9, 'weeeee', '2013-09-11 17:49:03', 0),
(17, 5, 9, 'weeeee', '2013-09-11 17:49:04', 0),
(18, 5, 9, 'weeeee', '2013-09-11 17:49:04', 0),
(19, 5, 9, 'weeeee', '2013-09-11 17:49:04', 0),
(20, 5, 9, 'weeeee', '2013-09-11 17:49:04', 0),
(42, 5, 9, 'sup', '2013-09-16 15:11:25', 0),
(43, 5, 9, 'Hi tosh', '2013-09-16 15:11:43', 0),
(44, 5, 9, 'Have you ever/ever felt like this?', '2013-09-16 15:16:37', 0),
(45, 5, 9, 'Long polling works now :O', '2013-09-16 15:26:51', 0),
(52, 5, 9, 'sup', '2013-09-16 17:58:23', 0),
(53, 5, 9, 'sup', '2013-09-16 17:58:30', 0),
(55, 5, 9, 'sup', '2013-09-16 18:36:33', 0),
(56, 5, 9, 'sup brah', '2013-09-16 18:36:47', 0),
(57, 5, 9, 'sup brah', '2013-09-16 18:41:25', 0),
(58, 5, 9, 'sup brah', '2013-09-16 19:16:15', 0),
(59, 5, 9, 'sup brah', '2013-09-16 19:58:15', 0),
(60, 5, 9, 'sasdfsadfsdfsafsdfasfa', '2013-09-17 11:19:13', 0),
(61, 5, 9, 'sasdfsadfsdfsafsdfasfa', '2013-09-17 11:19:18', 0),
(62, 5, 9, 'sasdfsadfsdfsafsdfasfa', '2013-09-17 11:21:12', 0),
(63, 80, 9, 'This is the API working!', '2013-09-19 13:38:46', 0),
(64, 80, 9, 'test', '2013-09-23 17:02:56', 0),
(65, 80, 9, 'test1', '2013-09-23 17:06:18', 0),
(66, 80, 9, 'test12', '2013-09-23 17:10:55', 0),
(67, 80, 9, 'test12', '2013-09-23 17:11:16', 0),
(68, 80, 9, 'test12', '2013-09-23 17:11:56', 0),
(69, 5, 9, 'elbereth', '2013-09-26 15:02:15', 0),
(70, 5, 9, 'elbereth', '2013-09-26 15:04:22', 0),
(71, 5, 9, 'elbereth1', '2013-09-26 15:04:55', 0),
(72, 5, 9, 'elbereth1', '2013-09-26 16:42:05', 0),
(73, 5, 9, 'elbereth1', '2013-09-26 16:42:07', 0),
(74, 5, 9, 'sup', '2013-10-07 17:42:40', 0),
(75, 5, 9, 'sup', '2013-10-07 17:43:56', 0),
(76, 5, 9, 'sup', '2013-10-07 17:44:04', 0),
(77, 67, 2395, 'sdf', '2013-10-09 14:54:59', 0),
(78, 67, 2395, 'sdf', '2013-10-09 14:57:29', 0),
(79, 67, 2395, 'sdf', '2013-10-09 14:57:31', 0),
(80, 67, 2395, 'sdf', '2013-10-09 14:57:32', 0),
(81, 67, 2395, 'cxvcx', '2013-10-09 16:26:39', 0),
(82, 67, 2395, 'cxvcx', '2013-10-09 16:26:41', 0),
(83, 67, 2395, 'Test', '2013-10-09 16:38:59', 0),
(84, 67, 2395, 'Last test', '2013-10-09 19:47:20', 0),
(85, 67, 9, 'Hi there. Can you see this?', '2013-10-09 20:34:55', 0);

-- --------------------------------------------------------

--
-- Table structure for table `conversations`
--

CREATE TABLE IF NOT EXISTS `conversations` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `creation_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `initiator` int(10) unsigned NOT NULL,
  `last_mod` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=453 ;

--
-- Dumping data for table `conversations`
--

INSERT INTO `conversations` (`id`, `creation_time`, `initiator`, `last_mod`) VALUES
(1, '2013-09-03 18:46:29', 9, '0000-00-00 00:00:00'),
(5, '2013-09-03 18:49:44', 9, '2013-10-10 14:07:43'),
(67, '2013-09-19 10:45:28', 2395, '2013-10-09 20:34:55'),
(81, '2013-09-19 12:40:20', 9, '2013-09-19 12:40:20'),
(451, '2013-10-10 16:35:51', 2395, '2013-10-10 16:35:51'),
(452, '2013-10-10 16:36:02', 2395, '2013-10-10 16:36:02');

-- --------------------------------------------------------

--
-- Table structure for table `conversation_participants`
--

CREATE TABLE IF NOT EXISTS `conversation_participants` (
  `conversation_id` int(10) unsigned NOT NULL,
  `participant_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `conversation_id` (`conversation_id`,`participant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- Dumping data for table `conversation_participants`
--

INSERT INTO `conversation_participants` (`conversation_id`, `participant_id`) VALUES
(5, 9),
(5, 1327),
(67, 646),
(67, 2395),
(80, 9),
(80, 21),
(81, 9),
(81, 820),
(451, 2229),
(451, 2395),
(452, 1595),
(452, 2395);

-- --------------------------------------------------------

--
-- Table structure for table `fb`
--

CREATE TABLE IF NOT EXISTS `fb` (
  `fb_id` varchar(64) COLLATE utf8_unicode_ci NOT NULL,
  `verified` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`fb_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

--
-- Dumping data for table `fb`
--


-- --------------------------------------------------------

--
-- Table structure for table `fb_invites`
--

CREATE TABLE IF NOT EXISTS `fb_invites` (
  `id` varchar(20) NOT NULL,
  `user_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `req` (`id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Dumping data for table `fb_invites`
--

INSERT INTO `fb_invites` (`id`, `user_id`) VALUES
('130709080412194', 9),
('521583667854914', 59);

-- --------------------------------------------------------

--
-- Table structure for table `logins`
--

CREATE TABLE IF NOT EXISTS `logins` (
  `user_id` int(10) unsigned NOT NULL,
  `logintime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY `user_id` (`user_id`,`logintime`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

--
-- Dumping data for table `logins`
--

INSERT INTO `logins` (`user_id`, `logintime`) VALUES
(9, '2013-02-07 17:52:07'),
(9, '2013-02-11 14:45:59'),
(2390, '2013-03-16 11:40:15');

-- --------------------------------------------------------

--
-- Table structure for table `network`
--

CREATE TABLE IF NOT EXISTS `network` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `parent` int(10) unsigned DEFAULT NULL,
  `cover_img` varchar(128) DEFAULT NULL,
  `is_university` tinyint(1) NOT NULL DEFAULT '0',
  `open_roulette` tinyint(1) NOT NULL DEFAULT '0',
  `open_marketplace` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 AUTO_INCREMENT=1911 ;

--
-- Dumping data for table `network`
--

INSERT INTO `network` (`id`, `name`, `parent`, `cover_img`, `is_university`, `open_roulette`, `open_marketplace`) VALUES
(1, 'University of Leeds', NULL, 'images/splash.png', 1, 0, 1),
(3, 'Leeds Metropolitan University', NULL, NULL, 1, 0, 0),
(5, 'University of Aberdeen', NULL, NULL, 1, 0, 0),
(9, 'University of Cambridge', NULL, NULL, 1, 0, 0),
(13, 'University of Newcastle upon Tyne', NULL, NULL, 1, 0, 0),

-- --------------------------------------------------------

--
-- Table structure for table `net_rules`
--

CREATE TABLE IF NOT EXISTS `net_rules` (
  `network_id` int(10) unsigned NOT NULL,
  `rule_type` varchar(10) NOT NULL,
  `rule_value` varchar(256) NOT NULL,
  UNIQUE KEY `network_id` (`network_id`,`rule_type`,`rule_value`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

--
-- Dumping data for table `net_rules`
--

INSERT INTO `net_rules` (`network_id`, `rule_type`, `rule_value`) VALUES
(1, 'email', 'leeds.ac.uk'),
(1, 'email', 'lmbru.ac.uk'),
(3, 'email', 'leeds-met.ac.uk'),
(3, 'email', 'leeds-metropolitan.ac.uk'),
(3, 'email', 'leedsmet.ac.uk'),
(3, 'email', 'leedsmetcarnegie.ac.uk'),
(3, 'email', 'leedsmetropolitan.ac.uk'),
(3, 'email', 'lmu.ac.uk'),
(5, 'email', 'abdn.ac.uk'),
(5, 'email', 'aberdeen.ac.uk'),
(6, 'email', 'darts.ac.uk'),
(9, 'email', 'cam.ac.uk'),
(9, 'email', 'cambridge-university.ac.uk'),
(9, 'email', 'cambridge.ac.uk'),
(9, 'email', 'cambridgeuniversity.ac.uk'),
(9, 'email', 'university-of-cambridge.ac.uk'),
(13, 'email', 'newcastle.ac.uk');

-- --------------------------------------------------------

--
-- Table structure for table `password_recovery`
--

CREATE TABLE IF NOT EXISTS `password_recovery` (
  `token` varchar(64) CHARACTER SET latin1 NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user` int(10) unsigned NOT NULL,
  PRIMARY KEY (`token`),
  UNIQUE KEY `users` (`user`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

--
-- Dumping data for table `password_recovery`
--

INSERT INTO `password_recovery` (`token`, `active`, `created_at`, `user`) VALUES
('eba3e7a960e7cfa86ca9f5f71434d909', 1, '2012-11-21 14:44:13', 9);

-- --------------------------------------------------------

--
-- Table structure for table `post_comments`
--

CREATE TABLE IF NOT EXISTS `post_comments` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int(10) unsigned NOT NULL,
  `by` int(10) unsigned NOT NULL,
  `text` varchar(1024) COLLATE utf8_bin NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=3 ;

--
-- Dumping data for table `post_comments`
--

INSERT INTO `post_comments` (`id`, `post_id`, `by`, `text`, `timestamp`) VALUES
(1, 1, 9, 'sup', '2013-09-09 20:12:37'),
(2, 1, 9, 'sup', '2013-09-09 20:12:40');

-- --------------------------------------------------------

--
-- Table structure for table `post_images`
--

CREATE TABLE IF NOT EXISTS `post_images` (
  `post_id` int(10) unsigned NOT NULL,
  `url` varchar(256) COLLATE utf8_bin NOT NULL,
  PRIMARY KEY (`post_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- Dumping data for table `post_images`
--

INSERT INTO `post_images` (`post_id`, `url`) VALUES
(11, 'http://dev.gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png'),
(36, 'http://dev.gleepost.com/uploads/35da2ca95be101a655961e37cc875b7b.png');

-- --------------------------------------------------------

--
-- Table structure for table `refers`
--

CREATE TABLE IF NOT EXISTS `refers` (
  `inviter` int(10) unsigned NOT NULL,
  `user` int(10) unsigned NOT NULL,
  PRIMARY KEY (`inviter`,`user`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Dumping data for table `refers`
--

INSERT INTO `refers` (`inviter`, `user`) VALUES
(305, 2171);

-- --------------------------------------------------------

--
-- Table structure for table `tokens`
--

CREATE TABLE IF NOT EXISTS `tokens` (
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(256) COLLATE utf8_bin NOT NULL,
  `expiry` datetime NOT NULL,
  PRIMARY KEY (`user_id`,`token`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

--
-- Dumping data for table `tokens`
--

INSERT INTO `tokens` (`user_id`, `token`, `expiry`) VALUES
(9, '3c1326c09c7d0111d19c770ad47932d57345f21e094923070b14e4b2cd39ecdb', '2013-09-10 15:02:36'),
(2395, '52bdda795c3c08d4f118f350b1965d02d7170ec9fa7e2172d6085eddaae5204c', '2013-09-19 18:22:12'),
(2395, 'e56e7924293f41002357977d7f9ab1678a9d3c41b142202540267896fbffa093', '2013-10-11 15:35:03'),
(2395, '5960e92225b94dbde0000c6571c234d2dd106d95151471de56a6dfcf36fcb673', '2013-10-11 15:35:45');

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE IF NOT EXISTS `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) CHARACTER SET latin1 NOT NULL,
  `password` varchar(64) CHARACTER SET latin1 NOT NULL,
  `email` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `is_admin` tinyint(1) NOT NULL DEFAULT '0',
  `is_banned` tinyint(1) NOT NULL DEFAULT '0',
  `avatar` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `location` int(10) unsigned DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `verified` tinyint(1) NOT NULL DEFAULT '0',
  `disabled` tinyint(1) NOT NULL DEFAULT '0',
  `fb` varchar(64) COLLATE utf8_unicode_ci DEFAULT NULL,
  `firstname` varchar(64) COLLATE utf8_unicode_ci DEFAULT NULL,
  `lastname` varchar(64) COLLATE utf8_unicode_ci DEFAULT NULL,
  `desc` varchar(256) COLLATE utf8_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=2398 ;

--
-- Dumping data for table `users`
--

INSERT INTO `users` (`id`, `name`, `password`, `email`, `is_admin`, `is_banned`, `avatar`, `location`, `timestamp`, `verified`, `disabled`, `fb`, `firstname`, `lastname`, `desc`) VALUES
(9, 'Patrick', '$2a$08$.2rwpDfhQLx3/p552gVjtOeKzug3DayPmmlsEcyFMo.uoybClJu1u', 'draaglom@gmail.com', 0, 0, 'uploads/bad2cbd1431260c2c4b9766ae5de25d6.gif', NULL, '2012-05-12 23:19:37', 1, 0, '1474356782', NULL, NULL, 'Foo. '),
(18, 'stefitou', '$2a$08$cUO3F5H2lHW.XjdSaHLShOc.XDDAb8/jTmdF2Gbf0V6A7KjDn05ia', 'cs11ss@leeds.ac.uk', 0, 0, NULL, NULL, '2012-05-12 23:19:37', 1, 0, NULL, NULL, NULL, NULL),
(21, 'petergatsby', '$2a$08$L3oxRGtt94j3bDiBBMCpQOkeg5JZObKg4GRDdspjXyDhiu1db3CZG', 'py11ooo@leeds.ac.uk', 0, 0, 'uploads/638e7b22f4091e0de5dbe5e110d2a6f9.gif', NULL, '2012-05-12 23:19:37', 1, 0, '662705958', NULL, NULL, 'Just do it.'),
(2365, 'tosh', '$2a$08$b9xobQo0VOpH8pKr/c62NunbygnvGNWLkqxfap/O7Fpi6SlbsiASu', 'tosh@leeds.ac.uk', 0, 0, NULL, NULL, '2012-11-27 01:29:24', 0, 0, NULL, NULL, NULL, NULL),
(2393, 'lukasz', '$2a$10$XYlYOIPMhLyhG/mvAUaMBO90b8Fx0ns35RE1qyEC9sGdsy8d1rX2C', 'asdasdsagsadgsagas@leeds.ac.uk', 0, 0, NULL, NULL, '2013-09-03 19:04:34', 0, 0, NULL, NULL, NULL, NULL),
(2394, 'Long polling works now :O', '$2a$10$qhXhkBbxMSIAb1aV5KfCA.mot5Lr4AAUONcdPuRML2TWGl1y1Pxwq', 'asdgeawgegasefae@leeds.ac.uk', 0, 0, NULL, NULL, '2013-09-16 15:48:52', 0, 0, NULL, NULL, NULL, NULL),
(2395, 'TestingUser', '$2a$10$ayDloCx7mFbbv9xH1MW.duoYw32y1xgWvGmE6I8KIAPXJNCcpwEC6', 'asdfsafasfasdfasdfsadfsadf@leeds.ac.uk', 0, 0, NULL, NULL, '2013-09-16 18:19:09', 0, 0, NULL, NULL, NULL, NULL);

-- --------------------------------------------------------

--
-- Table structure for table `user_network`
--

CREATE TABLE IF NOT EXISTS `user_network` (
  `user_id` int(10) unsigned NOT NULL,
  `network_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `user_id` (`user_id`,`network_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Dumping data for table `user_network`
--

INSERT INTO `user_network` (`user_id`, `network_id`) VALUES
(9, 1),
(18, 1),
(19, 1),
(21, 1),
(2365, 1),
(2395, 1);

-- --------------------------------------------------------

--
-- Table structure for table `verification`
--

CREATE TABLE IF NOT EXISTS `verification` (
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(128) CHARACTER SET latin1 NOT NULL,
  UNIQUE KEY `id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

--
-- Dumping data for table `verification`
--

INSERT INTO `verification` (`user_id`, `token`) VALUES
(9, '669c4e5970cd2fb727d79b81988d9d38'),
(21, '58828ac4af96ada832702aac67b526b8'),
(2390, '31f92971b24162f1ef81be5542abd299');

-- --------------------------------------------------------

--
-- Table structure for table `wall_comments`
--

CREATE TABLE IF NOT EXISTS `wall_comments` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int(10) unsigned NOT NULL,
  `by` int(10) unsigned NOT NULL,
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `text` varchar(1024) COLLATE utf8_bin NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=1 ;

--
-- Dumping data for table `wall_comments`
--


-- --------------------------------------------------------

--
-- Table structure for table `wall_posts`
--

CREATE TABLE IF NOT EXISTS `wall_posts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `by` int(10) unsigned NOT NULL,
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `text` varchar(1024) COLLATE utf8_bin NOT NULL,
  `network_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=41 ;

--
-- Dumping data for table `wall_posts`
--

INSERT INTO `wall_posts` (`id`, `by`, `time`, `text`, `network_id`) VALUES
(11, 9, '2013-09-18 19:09:18', 'sup brah', 1),
(24, 2395, '2013-10-09 16:19:05', 'asdfsafsadfasdfsadfasfd', 1),
(25, 2395, '2013-10-09 16:27:56', '', 1),
(31, 21, '2013-10-09 20:49:10', '', 1),
(34, 2395, '2013-10-09 21:12:15', 'Te1', 1),
(35, 2395, '2013-10-09 21:13:56', 'Fruits', 1),
(36, 2395, '2013-10-09 21:14:58', 'Post', 1),
(37, 21, '2013-10-09 22:02:32', 'I know the kind of girl that you are', 1),
(38, 2395, '2013-10-09 22:05:57', 'Oh yeah? What kind of girl am I???????', 1),
(39, 21, '2013-10-09 22:07:53', 'You''re DANGEROUS! She''s DANGEROUS. That girl is DANGEROUS', 1),
(40, 21, '2013-10-09 22:09:48', 'It might not be the right time. I might not be the right one, but there''s something about is I''ve got to say, cuz there''s something between us anyway.', 1);
