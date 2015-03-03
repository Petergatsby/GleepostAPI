-- phpMyAdmin SQL Dump
-- version 3.4.11.1deb2+deb7u1
-- http://www.phpmyadmin.net
--
-- Host: localhost
-- Generation Time: Feb 28, 2015 at 01:30 AM
-- Server version: 5.5.41
-- PHP Version: 5.4.4-14+deb7u5

SET SQL_MODE="NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;

--
-- Database: `gleepost`
--

-- --------------------------------------------------------

--
-- Table structure for table `categories`
--

CREATE TABLE IF NOT EXISTS `categories` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `tag` varchar(32) NOT NULL,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=latin1 AUTO_INCREMENT=13 ;

-- --------------------------------------------------------

--
-- Table structure for table `chat_messages`
--

CREATE TABLE IF NOT EXISTS `chat_messages` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `conversation_id` int(10) unsigned NOT NULL,
  `from` int(10) unsigned NOT NULL,
  `text` varchar(1024) DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `seen` tinyint(1) NOT NULL DEFAULT '0',
  `system` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4 AUTO_INCREMENT=9112 ;

-- --------------------------------------------------------

--
-- Table structure for table `contacts`
--

CREATE TABLE IF NOT EXISTS `contacts` (
  `adder` int(10) unsigned NOT NULL,
  `addee` int(10) unsigned NOT NULL,
  `confirmed` tinyint(1) NOT NULL DEFAULT '0',
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `accepted` timestamp NULL DEFAULT NULL,
  UNIQUE KEY `contacted` (`adder`,`addee`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Table structure for table `contact_requests`
--

CREATE TABLE IF NOT EXISTS `contact_requests` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `full_name` varchar(255) COLLATE utf8_bin NOT NULL,
  `college` varchar(255) COLLATE utf8_bin NOT NULL,
  `email` varchar(255) COLLATE utf8_bin NOT NULL,
  `phone_no` varchar(255) COLLATE utf8_bin NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=19 ;

-- --------------------------------------------------------

--
-- Table structure for table `conversations`
--

CREATE TABLE IF NOT EXISTS `conversations` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `creation_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `initiator` int(10) unsigned NOT NULL,
  `primary_conversation` tinyint(1) NOT NULL DEFAULT '0',
  `group_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=4552 ;

-- --------------------------------------------------------

--
-- Table structure for table `conversation_participants`
--

CREATE TABLE IF NOT EXISTS `conversation_participants` (
  `conversation_id` int(10) unsigned NOT NULL,
  `participant_id` int(10) unsigned NOT NULL,
  `last_read` int(10) unsigned NOT NULL DEFAULT '0',
  `deleted` tinyint(1) NOT NULL DEFAULT '0',
  `deletion_threshold` int(10) unsigned NOT NULL DEFAULT '0',
  UNIQUE KEY `conversation_id` (`conversation_id`,`participant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Table structure for table `devices`
--

CREATE TABLE IF NOT EXISTS `devices` (
  `user_id` int(10) unsigned NOT NULL,
  `device_type` varchar(64) COLLATE utf8_bin NOT NULL,
  `device_id` varchar(255) COLLATE utf8_bin NOT NULL,
  `last_update` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `application` varchar(100) COLLATE utf8_bin NOT NULL DEFAULT 'gleepost',
  UNIQUE KEY `u_d` (`user_id`,`device_id`),
  UNIQUE KEY `device_id` (`device_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Table structure for table `event_attendees`
--

CREATE TABLE IF NOT EXISTS `event_attendees` (
  `post_id` int(10) unsigned NOT NULL,
  `user_id` int(10) unsigned NOT NULL,
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `post_id` (`post_id`,`user_id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `facebook`
--

CREATE TABLE IF NOT EXISTS `facebook` (
  `fb_id` bigint(20) unsigned NOT NULL,
  `user_id` int(10) unsigned DEFAULT NULL,
  `email` varchar(200) DEFAULT NULL,
  UNIQUE KEY `fb_id` (`fb_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `facebook_verification`
--

CREATE TABLE IF NOT EXISTS `facebook_verification` (
  `fb_id` bigint(20) unsigned NOT NULL,
  `token` varchar(255) NOT NULL,
  UNIQUE KEY `fb_id` (`fb_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `fb`
--

CREATE TABLE IF NOT EXISTS `fb` (
  `fb_id` varchar(64) COLLATE utf8_unicode_ci NOT NULL,
  `verified` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`fb_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `fb_group_invites`
--

CREATE TABLE IF NOT EXISTS `fb_group_invites` (
  `inviter_user_id` int(10) unsigned NOT NULL,
  `facebook_id` bigint(32) unsigned NOT NULL,
  `network_id` int(10) unsigned NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `accepted` tinyint(1) NOT NULL DEFAULT '0',
  KEY `fb` (`facebook_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `fb_invites`
--

CREATE TABLE IF NOT EXISTS `fb_invites` (
  `id` varchar(20) NOT NULL,
  `user_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `req` (`id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `goose_db_version`
--

CREATE TABLE IF NOT EXISTS `goose_db_version` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint(20) NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=latin1 AUTO_INCREMENT=32 ;

--
-- Dumping data for table `goose_db_version`
--

INSERT INTO `goose_db_version` (`id`, `version_id`, `is_applied`, `tstamp`) VALUES
(1, 0, 1, '2014-10-31 21:57:26'),
(2, 20141031150208, 1, '2014-10-31 22:26:52'),
(3, 20141031150208, 0, '2014-10-31 22:27:02'),
(4, 20141031150208, 1, '2014-10-31 22:27:30'),
(5, 20141104165143, 1, '2014-11-05 23:10:37'),
(6, 20141104165143, 0, '2014-11-05 23:13:26'),
(7, 20141104165143, 1, '2014-11-05 23:15:52'),
(8, 20141105180258, 1, '2014-11-06 19:53:05'),
(9, 20141106120249, 1, '2014-11-06 21:17:35'),
(10, 20141107160321, 1, '2014-11-08 00:21:14'),
(11, 20141111121859, 1, '2014-11-11 20:32:39'),
(12, 20141118154734, 1, '2014-11-20 00:19:14'),
(13, 20141119144338, 1, '2014-11-20 00:19:24'),
(14, 20141120154940, 1, '2014-11-21 00:10:24'),
(15, 20141120163306, 1, '2014-11-21 00:37:18'),
(16, 20141125170337, 1, '2014-11-26 02:02:21'),
(17, 20141216221609, 1, '2014-12-18 19:00:08'),
(18, 20150116222816, 1, '2015-01-17 03:33:32'),
(19, 20150122160128, 1, '2015-01-22 21:09:23'),
(20, 20150128152252, 1, '2015-01-28 20:28:55'),
(21, 20150130201806, 1, '2015-02-02 22:57:26'),
(22, 20150202143600, 1, '2015-02-03 00:54:30'),
(23, 20150204181648, 1, '2015-02-06 00:51:07'),
(24, 20150205160341, 1, '2015-02-06 00:58:22'),
(25, 20150217123826, 1, '2015-02-17 20:43:41'),
(26, 20150218132731, 1, '2015-02-18 21:36:23'),
(27, 20150218134144, 1, '2015-02-18 21:44:27'),
(28, 20150224145928, 1, '2015-02-24 23:38:44'),
(29, 20150224174100, 1, '2015-02-25 01:44:47'),
(30, 20150225162015, 1, '2015-02-26 00:24:03'),
(31, 20150302154058, 1, '2015-03-02 23:48:54');

-- --------------------------------------------------------

--
-- Table structure for table `group_invites`
--

CREATE TABLE IF NOT EXISTS `group_invites` (
  `group_id` int(10) unsigned NOT NULL,
  `inviter` int(10) unsigned NOT NULL,
  `email` varchar(128) NOT NULL,
  `key` varchar(256) NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `accepted` tinyint(1) NOT NULL DEFAULT '0',
  KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `interest`
--

CREATE TABLE IF NOT EXISTS `interest` (
  `email` varchar(128) NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `invite`
--

CREATE TABLE IF NOT EXISTS `invite` (
  `user` int(11) NOT NULL,
  `token` varchar(256) NOT NULL,
  `count` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`user`,`token`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `invites`
--

CREATE TABLE IF NOT EXISTS `invites` (
  `code` varchar(32) CHARACTER SET latin1 NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' ON UPDATE CURRENT_TIMESTAMP,
  `used` tinyint(1) NOT NULL DEFAULT '0',
  `sent_to` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  PRIMARY KEY (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `listings`
--

CREATE TABLE IF NOT EXISTS `listings` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `desc` text CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  `owner` int(10) NOT NULL,
  `where` int(10) unsigned NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `type` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `timestamp` (`timestamp`),
  FULLTEXT KEY `desc` (`desc`),
  FULLTEXT KEY `ft` (`title`,`desc`)
) ENGINE=MyISAM  DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=939 ;

-- --------------------------------------------------------

--
-- Table structure for table `listing_attribs`
--

CREATE TABLE IF NOT EXISTS `listing_attribs` (
  `listing` int(10) unsigned NOT NULL,
  `attrib` varchar(64) CHARACTER SET latin1 NOT NULL,
  `value` varchar(512) CHARACTER SET latin1 NOT NULL,
  PRIMARY KEY (`listing`,`attrib`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `listing_disable`
--

CREATE TABLE IF NOT EXISTS `listing_disable` (
  `listing_id` int(10) unsigned NOT NULL,
  `disable_type` int(5) NOT NULL,
  `disable_reason` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`listing_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `listing_images`
--

CREATE TABLE IF NOT EXISTS `listing_images` (
  `listing` int(10) unsigned NOT NULL,
  `path` varchar(128) CHARACTER SET latin1 NOT NULL,
  PRIMARY KEY (`listing`,`path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `listing_network`
--

CREATE TABLE IF NOT EXISTS `listing_network` (
  `listing_id` int(10) unsigned NOT NULL,
  `network_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`listing_id`,`network_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `listing_types`
--

CREATE TABLE IF NOT EXISTS `listing_types` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) CHARACTER SET latin1 NOT NULL,
  `parent` int(10) unsigned NOT NULL DEFAULT '0',
  `neatname` varchar(64) COLLATE utf8_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=61 ;

-- --------------------------------------------------------

--
-- Table structure for table `location`
--

CREATE TABLE IF NOT EXISTS `location` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `desc` varchar(256) CHARACTER SET latin1 NOT NULL,
  `line_1` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `line_2` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `line_3` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `postcode` varchar(10) CHARACTER SET latin1 DEFAULT NULL,
  `country` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `lat` float DEFAULT NULL,
  `long` float DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=1 ;

-- --------------------------------------------------------

--
-- Table structure for table `logins`
--

CREATE TABLE IF NOT EXISTS `logins` (
  `user_id` int(10) unsigned NOT NULL,
  `logintime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY `user_id` (`user_id`,`logintime`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `messages`
--

CREATE TABLE IF NOT EXISTS `messages` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `from` int(10) unsigned NOT NULL,
  `to` int(10) NOT NULL,
  `subject` varchar(128) CHARACTER SET latin1 NOT NULL DEFAULT '[No Subject]',
  `body` text CHARACTER SET latin1 NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `read` bit(1) NOT NULL DEFAULT b'0',
  `conv_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=381 ;

-- --------------------------------------------------------

--
-- Table structure for table `network`
--

CREATE TABLE IF NOT EXISTS `network` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `parent` int(10) unsigned DEFAULT NULL,
  `cover_img` varchar(128) DEFAULT NULL,
  `desc` varchar(255) DEFAULT NULL,
  `creator` int(10) unsigned DEFAULT NULL,
  `is_university` tinyint(1) NOT NULL DEFAULT '0',
  `open_roulette` tinyint(1) NOT NULL DEFAULT '0',
  `open_marketplace` tinyint(1) NOT NULL DEFAULT '0',
  `user_group` tinyint(1) NOT NULL DEFAULT '0',
  `privacy` varchar(20) DEFAULT 'private',
  `master_group` int(10) unsigned DEFAULT NULL,
  `approval_level` int(5) unsigned NOT NULL DEFAULT '0',
  `approved_categories` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 AUTO_INCREMENT=2321 ;

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

-- --------------------------------------------------------

--
-- Table structure for table `notifications`
--

CREATE TABLE IF NOT EXISTS `notifications` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` varchar(256) COLLATE utf8_bin NOT NULL,
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `recipient` int(10) unsigned NOT NULL,
  `by` int(10) unsigned NOT NULL,
  `post_id` int(10) unsigned DEFAULT NULL,
  `seen` tinyint(1) NOT NULL DEFAULT '0',
  `network_id` int(10) unsigned DEFAULT NULL,
  `preview_text` varchar(100) COLLATE utf8_bin DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=3882 ;

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

-- --------------------------------------------------------

--
-- Table structure for table `post_attribs`
--

CREATE TABLE IF NOT EXISTS `post_attribs` (
  `post_id` int(10) unsigned NOT NULL,
  `attrib` varchar(64) CHARACTER SET utf8 NOT NULL,
  `value` varchar(512) CHARACTER SET utf8 NOT NULL,
  PRIMARY KEY (`post_id`,`attrib`),
  KEY `value` (`value`(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `post_categories`
--

CREATE TABLE IF NOT EXISTS `post_categories` (
  `post_id` int(10) unsigned NOT NULL,
  `category_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `post_id` (`post_id`,`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `post_comments`
--

CREATE TABLE IF NOT EXISTS `post_comments` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int(10) unsigned NOT NULL,
  `by` int(10) unsigned NOT NULL,
  `text` varchar(1024) DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4 AUTO_INCREMENT=1488 ;

-- --------------------------------------------------------

--
-- Table structure for table `post_images`
--

CREATE TABLE IF NOT EXISTS `post_images` (
  `post_id` int(10) unsigned NOT NULL,
  `url` varchar(256) COLLATE utf8_bin NOT NULL,
  KEY `postid` (`post_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Table structure for table `post_likes`
--

CREATE TABLE IF NOT EXISTS `post_likes` (
  `post_id` int(10) unsigned NOT NULL,
  `user_id` int(10) unsigned NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `post_id` (`post_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `post_reviews`
--

CREATE TABLE IF NOT EXISTS `post_reviews` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int(10) unsigned NOT NULL,
  `action` varchar(255) COLLATE utf8_bin NOT NULL,
  `by` int(10) unsigned NOT NULL,
  `reason` varchar(1024) COLLATE utf8_bin DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=290 ;

-- --------------------------------------------------------

--
-- Table structure for table `post_videos`
--

CREATE TABLE IF NOT EXISTS `post_videos` (
  `post_id` int(10) unsigned NOT NULL,
  `video_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `post_id` (`post_id`,`video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `post_views`
--

CREATE TABLE IF NOT EXISTS `post_views` (
  `user_id` int(10) unsigned NOT NULL,
  `post_id` int(10) unsigned NOT NULL,
  `ts` datetime NOT NULL,
  KEY `p_t` (`post_id`,`ts`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

-- --------------------------------------------------------

--
-- Table structure for table `refers`
--

CREATE TABLE IF NOT EXISTS `refers` (
  `inviter` int(10) unsigned NOT NULL,
  `user` int(10) unsigned NOT NULL,
  PRIMARY KEY (`inviter`,`user`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `reports`
--

CREATE TABLE IF NOT EXISTS `reports` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `userip` varchar(255) NOT NULL,
  `reporteruserip` varchar(255) NOT NULL,
  `timestamp` datetime NOT NULL,
  `username` varchar(255) NOT NULL,
  `reporterusername` varchar(255) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=latin1 AUTO_INCREMENT=3 ;

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

-- --------------------------------------------------------

--
-- Table structure for table `types`
--

CREATE TABLE IF NOT EXISTS `types` (
  `listing_id` int(10) unsigned NOT NULL,
  `type_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `listing_type` (`listing_id`,`type_id`),
  KEY `type_id` (`type_id`),
  KEY `listings` (`listing_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `type_attribs`
--

CREATE TABLE IF NOT EXISTS `type_attribs` (
  `type_id` int(10) unsigned NOT NULL,
  `attrib` varchar(64) CHARACTER SET latin1 NOT NULL,
  `input_type` varchar(64) CHARACTER SET latin1 NOT NULL,
  `full_name` varchar(64) CHARACTER SET latin1 NOT NULL,
  PRIMARY KEY (`type_id`,`attrib`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `uploads`
--

CREATE TABLE IF NOT EXISTS `uploads` (
  `user_id` int(10) unsigned NOT NULL,
  `url` varchar(256) COLLATE utf8_bin DEFAULT NULL,
  `upload_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `type` varchar(10) COLLATE utf8_bin NOT NULL DEFAULT 'image',
  `status` varchar(15) COLLATE utf8_bin DEFAULT NULL,
  `mp4_url` varchar(255) COLLATE utf8_bin DEFAULT NULL,
  `webm_url` varchar(255) COLLATE utf8_bin DEFAULT NULL,
  `upload_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  UNIQUE KEY `upload_id` (`upload_id`),
  KEY `user` (`user_id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=3916 ;

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE IF NOT EXISTS `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `password` varchar(64) CHARACTER SET latin1 NOT NULL,
  `email` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `is_admin` tinyint(1) NOT NULL DEFAULT '0',
  `is_banned` tinyint(1) NOT NULL DEFAULT '0',
  `avatar` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL,
  `location` int(10) unsigned DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `verified` tinyint(1) NOT NULL DEFAULT '0',
  `disabled` tinyint(1) NOT NULL DEFAULT '0',
  `fb` varchar(64) COLLATE utf8_unicode_ci DEFAULT NULL,
  `firstname` varchar(64) COLLATE utf8_unicode_ci NOT NULL,
  `lastname` varchar(64) COLLATE utf8_unicode_ci DEFAULT NULL,
  `desc` varchar(256) COLLATE utf8_unicode_ci DEFAULT NULL,
  `busy` tinyint(1) NOT NULL DEFAULT '0',
  `official` tinyint(1) NOT NULL DEFAULT '0',
  `new_message_threshold` datetime NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=2913 ;

-- --------------------------------------------------------

--
-- Table structure for table `user_at`
--

CREATE TABLE IF NOT EXISTS `user_at` (
  `user_id` int(10) unsigned NOT NULL,
  `address_id` int(10) unsigned NOT NULL,
  UNIQUE KEY `user_address` (`user_id`,`address_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `user_network`
--

CREATE TABLE IF NOT EXISTS `user_network` (
  `user_id` int(10) unsigned NOT NULL,
  `network_id` int(10) unsigned NOT NULL,
  `role` varchar(16) NOT NULL DEFAULT 'member',
  `role_level` int(2) NOT NULL DEFAULT '1',
  UNIQUE KEY `user_id` (`user_id`,`network_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `user_reports`
--

CREATE TABLE IF NOT EXISTS `user_reports` (
  `reporter_id` int(10) unsigned NOT NULL,
  `type` varchar(100) NOT NULL,
  `entity_id` int(10) unsigned NOT NULL,
  `reason` varchar(255) DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `report` (`reporter_id`,`type`,`entity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- Table structure for table `verification`
--

CREATE TABLE IF NOT EXISTS `verification` (
  `user_id` int(10) unsigned NOT NULL,
  `token` varchar(128) CHARACTER SET latin1 NOT NULL,
  UNIQUE KEY `id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `videochat_log`
--

CREATE TABLE IF NOT EXISTS `videochat_log` (
  `sessionid` varchar(256) COLLATE utf8_bin NOT NULL,
  `tx` int(10) unsigned NOT NULL,
  `rx` int(10) unsigned DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `message` varchar(2000) COLLATE utf8_bin NOT NULL,
  KEY `timestamp` (`timestamp`),
  KEY `sessionid` (`sessionid`(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

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

-- --------------------------------------------------------

--
-- Table structure for table `wall_posts`
--

CREATE TABLE IF NOT EXISTS `wall_posts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `by` int(10) unsigned NOT NULL,
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `text` varchar(1024) DEFAULT NULL,
  `network_id` int(10) unsigned NOT NULL,
  `deleted` tinyint(1) NOT NULL DEFAULT '0',
  `pending` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4 AUTO_INCREMENT=2771 ;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
