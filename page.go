package main

import (
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/inflect"
	"github.com/tidusant/c3m-common/log"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"

	//	"c3m/common/inflect"
	//	"c3m/log"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"strings"
)

const (
	defaultcampaigncode string = "XVsdAZGVmY"
)

type Arith int

func (t *Arith) Run(data string, result *models.RequestResult) error {

	*result = models.RequestResult{}
	//parse args
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}

	//	if usex.Action == "c" {
	//		*result = CreateProduct(usex)

	//	} else

	usex.Shop = rpch.GetShopById(usex.UserID, ShopID)
	if usex.Shop.ID.Hex() == "" {
		*result = c3mcommon.ReturnJsonMessage("0", "shop not found", "", "")
		return nil
	}

	if usex.Action == "s" {
		*result = SavePage(usex)
	} else if usex.Action == "l" {
		*result = LoadPage(usex)
	} else if usex.Action == "la" {
		*result = LoadAllPageCode(usex)
	}

	return nil
}

// func Remove(usex models.UserSession) string {
// 	log.Debugf("remove  %s", usex.Params)
// 	args := strings.Split(usex.Params, ",")
// 	if len(args) < 2 {
// 		return c3mcommon.ReturnJsonMessage("0", "error submit fields", "", "")
// 	}
// 	log.Debugf("save prod %s", args)
// 	code := args[0]
// 	lang := args[1]
// 	itemremove := rpch.GetPageByCode(usex.UserID, usex.ShopID, code)
// 	if itemremove.Langs[lang] != nil {
// 		//remove slug
// 		rpch.RemoveSlug(itemremove.Langs[lang].Slug, usex.ShopID)
// 		delete(itemremove.Langs, lang)
// 		rpch.SavePage(itemremove)
// 	}

// 	//build home
// 	var bs models.BuildScript
// 	shop := rpch.GetShopById(usex.UserID, usex.ShopID)
// 	bs.ShopID = usex.ShopID
// 	bs.TemplateCode = shop.Theme
// 	bs.Domain = shop.Domain
// 	bs.ObjectID = "home"
// 	rpb.CreateBuild(bs)

// 	//build cat
// 	bs.Collection = "page"
// 	bs.ObjectID = itemremove.Code
// 	rpb.CreateBuild(bs)
// 	return c3mcommon.ReturnJsonMessage("1", "", "success", "")

// }
func LoadPage(usex models.UserSession) models.RequestResult {
	args := strings.Split(usex.Params, ",")
	if len(args) < 1 {
		return c3mcommon.ReturnJsonMessage("0", "error submit fields", "", "")
	}

	code := args[0]
	item := rpch.GetPageByCode(usex.Shop.Theme, usex.Shop.ID.Hex(), code)
	//output data
	rs := struct {
		Code   string
		Langs  map[string]models.PageLang
		Seo    string
		Blocks []models.PageBlock
	}{
		item.Code,
		item.Langs,
		item.Seo,
		item.Blocks,
	}
	b, _ := json.Marshal(rs)

	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))

}
func LoadAllPageCode(usex models.UserSession) models.RequestResult {

	items := rpch.GetAllPageCode(usex.Shop.Theme, usex.Shop.ID.Hex())
	if len(items) == 0 {
		return c3mcommon.ReturnJsonMessage("2", "", "no page found", "")
	}
	b, _ := json.Marshal(items)
	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))

}
func SavePage(usex models.UserSession) models.RequestResult {
	var newitem models.Page
	log.Debugf("Unmarshal %s", usex.Params)
	err := json.Unmarshal([]byte(usex.Params), &newitem)
	if !c3mcommon.CheckError("json parse page", err) {
		return c3mcommon.ReturnJsonMessage("0", "json parse fail", "", "")
	}

	//update
	//check olditem
	log.Debugf("json parse: %v", newitem)
	olditem := rpch.GetPageByCode(usex.Shop.Theme, usex.Shop.ID.Hex(), newitem.Code)
	if olditem.Code == "" {
		return c3mcommon.ReturnJsonMessage("0", "item not found", "", "")
	}

	var langlinks []models.LangLink
	for langcode, _ := range newitem.Langs {
		if newitem.Langs[langcode].Title == "" {
			continue
		}
		var newslug models.Slug
		newslug.ShopId = usex.Shop.ID.Hex()
		newslug.Object = "page"
		newslug.ObjectId = olditem.ID.Hex()
		newslug.Lang = langcode
		newslug.TemplateCode = usex.Shop.Theme

		//newslug
		pagelang := newitem.Langs[langcode]
		log.Debugf("pagelang %v", pagelang)
		newslug.Slug = inflect.ParameterizeJoin(newitem.Langs[langcode].Title, "_")
		//check slug
		if newitem.Code == "home" && usex.Shop.Config.DefaultLang == langcode {
			newslug.Slug = ""
		}

		pagelang.Slug = rpch.SaveSlug(newslug)
		newitem.Langs[langcode] = pagelang

		if newitem.Langs[langcode].Slug != "" {
			langlinks = append(langlinks, models.LangLink{Href: newitem.Langs[langcode].Slug + "/", Code: langcode, Name: c3mcommon.GetLangnameByCode(langcode)})
		} else {
			langlinks = append(langlinks, models.LangLink{Href: newitem.Langs[langcode].Slug, Code: langcode, Name: c3mcommon.GetLangnameByCode(langcode)})
		}
		//=====

	}

	//update
	olditem.Seo = newitem.Seo
	olditem.Langs = newitem.Langs
	olditem.Blocks = newitem.Blocks
	olditem.LangLinks = langlinks
	strrt := rpch.SavePage(olditem)
	if strrt == "0" {
		return c3mcommon.ReturnJsonMessage("0", "error", "error", "")
	}

	//rebuild page
	b, err := json.Marshal(olditem)

	//create build

	errstr := rpch.CreateBuild("page", olditem.ID.Hex(), string(b), usex)
	if errstr != "" {
		return c3mcommon.ReturnJsonMessage("0", errstr, "build error", "")
	}
	errstr = rpch.CreateCommonDataBuild(usex)
	if errstr != "" {
		return c3mcommon.ReturnJsonMessage("0", errstr, "build error", "")
	}

	// //rebuild home
	// var bs models.BuildScript
	// shop := rpch.GetShopById(usex.UserID, usex.Shop.ID.Hex())
	// bs.ShopID = usex.Shop.ID.Hex()
	// bs.TemplateCode = shop.Theme
	// bs.Domain = shop.Domain
	//  bs.ObjectID = "home"
	//  rpb.CreateBuild(bs)

	// //rebuild cat
	// bs.Collection = "page"
	// bs.ObjectID = newitem.Code
	// rpb.CreateBuild(bs)
	return c3mcommon.ReturnJsonMessage("1", "", "success", "")
}

func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9883, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	logLevel := log.DebugLevel
	if !debug {
		logLevel = log.InfoLevel

	}

	log.SetOutputFile(fmt.Sprintf("page-"+strconv.Itoa(port)), logLevel)
	defer log.CloseOutputFile()
	log.RedirectStdOut()

	//init db
	arith := new(Arith)
	rpc.Register(arith)
	log.Infof("running with port:" + strconv.Itoa(port))

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	c3mcommon.CheckError("rpc dail:", err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	c3mcommon.CheckError("rpc init listen", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}
