package main

import (
	"github.com/nlopes/slack"
)

// fmtSendHelp does what it sounds like
func fmtSendHelp() slack.MsgOption {

	divSection := slack.NewDividerBlock()
	headerText := slack.NewTextBlockObject("mrkdwn", "*help*", false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	divSection = slack.NewDividerBlock()
	opslogText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog <entry> [#<tag>:<value>]...*\n\tCreate a new opslog entry, optionally adding tags",
		false,
		false,
	)
	opslogSection := slack.NewSectionBlock(opslogText, nil, nil)
	deleteText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog deletelast*\n\tDelete the previous opslog entry created by you",
		false,
		false,
	)
	deleteSection := slack.NewSectionBlock(deleteText, nil, nil)
	showText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog show [x]*\n\tList the previous x opslog entries in the channel it's called from, defaults to 10",
		false,
		false,
	)
	showSection := slack.NewSectionBlock(showText, nil, nil)
	showAllText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog showall [x]*\n\tList the previous x opslog entries globally, defaults to 10",
		false,
		false,
	)
	showAllSection := slack.NewSectionBlock(showAllText, nil, nil)
	searchText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog search <entry>*\n\tSearch for opslog entries in the channel it's called from, limited to 50 results",
		false,
		false,
	)
	searchSection := slack.NewSectionBlock(searchText, nil, nil)
	searchAllText := slack.NewTextBlockObject(
		"mrkdwn",
		"*/opslog searchall <entry>*\n\tSearch for opslog entries globally, limited to 50 results",
		false,
		false,
	)
	searchAllSection := slack.NewSectionBlock(searchAllText, nil, nil)

	msg := slack.MsgOptionBlocks(
		divSection,
		headerSection,
		divSection,
		opslogSection,
		deleteSection,
		showSection,
		showAllSection,
		searchSection,
		searchAllSection,
	)

	return msg
}
