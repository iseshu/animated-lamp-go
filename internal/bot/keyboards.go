package bot

import (
	"fmt"

	tele "gopkg.in/telebot.v3"

	"animated-lamp-go/internal/config"
	"animated-lamp-go/internal/storage"
)

type Keyboards struct {
	Menu *tele.ReplyMarkup

	// Action buttons
	BtnScreenshot2  tele.Btn
	BtnScreenshot4  tele.Btn
	BtnScreenshot6  tele.Btn
	BtnScreenshot8  tele.Btn
	BtnScreenshot10 tele.Btn

	BtnManualScreenshot tele.Btn
	BtnSampleVideo      tele.Btn
	BtnTrimVideo        tele.Btn
	BtnMediaInfo        tele.Btn

	// Settings buttons
	BtnToggleUploadMode tele.Btn
	BtnToggleMode       tele.Btn
	BtnWatermarkText    tele.Btn
	BtnWatermarkColor   tele.Btn
	BtnWatermarkPos     tele.Btn
	BtnWatermarkFont    tele.Btn
	BtnSampleDuration   tele.Btn
	BtnCloseSettings    tele.Btn
}

func NewKeyboards() *Keyboards {
	menu := &tele.ReplyMarkup{}
	return &Keyboards{
		Menu: menu,

		BtnScreenshot2:  menu.Data("2", "scht_2"),
		BtnScreenshot4:  menu.Data("4", "scht_4"),
		BtnScreenshot6:  menu.Data("6", "scht_6"),
		BtnScreenshot8:  menu.Data("8", "scht_8"),
		BtnScreenshot10: menu.Data("10", "scht_10"),

		BtnManualScreenshot: menu.Data("🎯 Manual Screenshots", "act_manual"),
		BtnSampleVideo:      menu.Data("🎞 Sample Video", "act_sample"),
		BtnTrimVideo:        menu.Data("✂️ Trim Video", "act_trim"),
		BtnMediaInfo:        menu.Data("ℹ️ Media Info", "act_mediainfo"),

		BtnToggleUploadMode: menu.Data("Upload Mode", "set_upload_mode"),
		BtnToggleMode:       menu.Data("Screenshot Mode", "set_scht_mode"),
		BtnWatermarkText:    menu.Data("Watermark Text", "set_wm_text"),
		BtnWatermarkColor:   menu.Data("Watermark Color", "set_wm_color"),
		BtnWatermarkPos:     menu.Data("Watermark Position", "set_wm_pos"),
		BtnWatermarkFont:    menu.Data("Font Size", "set_wm_font"),
		BtnSampleDuration:   menu.Data("Sample Duration", "set_smpl_dur"),
		BtnCloseSettings:    menu.Data("❌ Close", "set_close"),
	}
}

func (k *Keyboards) ActionsMenu(duration int) *tele.ReplyMarkup {
	menu := k.Menu

	row1 := menu.Row(k.BtnScreenshot2, k.BtnScreenshot4, k.BtnScreenshot6, k.BtnScreenshot8, k.BtnScreenshot10)
	row2 := menu.Row(k.BtnManualScreenshot, k.BtnMediaInfo)
	row3 := menu.Row(k.BtnTrimVideo)

	if duration >= 60 {
		row3 = menu.Row(k.BtnSampleVideo, k.BtnTrimVideo)
	}

	menu.Inline(row1, row2, row3)
	return menu
}

func (k *Keyboards) SettingsMenu(u *storage.User) (*tele.ReplyMarkup, string) {
	menu := k.Menu

	uploadModeStr := "🖼 Photo"
	if u.AsFile {
		uploadModeStr = "📁 Document"
	}

	modeStr := "⚖️ Equal Interval"
	if u.ScreenshotMode == 1 {
		modeStr = "🎲 Random"
	}

	colorStr := config.Colors[u.WatermarkColor]
	posStr := config.Positions[u.WatermarkPosition]
	fontStr := config.FontSizesName[u.FontSize]

	wmText := u.WatermarkText
	if wmText == "" {
		wmText = "Disabled"
	}

	btnUpload := menu.Data(fmt.Sprintf("Upload: %s", uploadModeStr), "set_upload_mode")
	btnMode := menu.Data(fmt.Sprintf("Mode: %s", modeStr), "set_scht_mode")
	btnColor := menu.Data(fmt.Sprintf("Color: %s", colorStr), "set_wm_color")
	btnPos := menu.Data(fmt.Sprintf("Position: %s", posStr), "set_wm_pos")
	btnFont := menu.Data(fmt.Sprintf("Font: %s", fontStr), "set_wm_font")
	btnDur := menu.Data(fmt.Sprintf("Sample: %ds", u.SampleDuration), "set_smpl_dur")
	btnText := menu.Data(fmt.Sprintf("Watermark: %s", wmText), "set_wm_text")
	btnClose := menu.Data("❌ Close Settings", "set_close")

	menu.Inline(
		menu.Row(btnUpload, btnMode),
		menu.Row(btnText),
		menu.Row(btnColor, btnPos),
		menu.Row(btnFont, btnDur),
		menu.Row(btnClose),
	)

	text := fmt.Sprintf("⚙️ **Bot Settings**\n\n"+
		"• **Upload Mode:** %s\n"+
		"• **Screenshot Mode:** %s\n"+
		"• **Watermark Text:** `%s`\n"+
		"• **Color:** %s | **Size:** %s\n"+
		"• **Position:** %s\n"+
		"• **Sample Duration:** %d seconds",
		uploadModeStr, modeStr, wmText, colorStr, fontStr, posStr, u.SampleDuration)

	return menu, text
}
