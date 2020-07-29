package handler

import (
	"fmt"
	"strconv"

	"memoapp/internal/database"
	"memoapp/internal/types"
	"memoapp/model"

	"github.com/emicklei/go-restful/log"
	"github.com/labstack/echo/v4"
)

type (
	// MemoAppOutput レスポンス用のデータ型
	MemoAppOutput struct {
		Memos   types.Memos
		Message string
	}

	// MemoHandler メモ用ハンドラー
	MemoHandler struct {
		HasCache bool
		repo     database.Database
		echo     *echo.Echo
	}

	endPointHandler func(c echo.Context) ([]byte, error)
)

var (
	pkgName = "handler"
)

// ProvideHandler メモハンドラーからルーティングを設定する
func ProvideHandler(e *echo.Echo) *MemoHandler {
	hdr := &MemoHandler{echo: e}
	routes := []struct {
		method     string
		path       string
		handler    endPointHandler
		cache      bool
		cacheClear bool
	}{
		{
			"GET",
			"/list",
			hdr.MemoIndex,
			true,
			false,
		},
		{
			"POST",
			"/",
			hdr.MemoCreate,
			false,
			true,
		},
		{
			"DELETE",
			"/:id",
			hdr.MemoDelete,
			false,
			true,
		},
	}
	for _, r := range routes {
		if r.cache {
			e.Add(r.method, r.path, hdr.cacheEndpointHandler(r.handler))
		} else {
			e.Add(r.method, r.path, hdr.endpointHandler(r.handler, r.cacheClear))
		}
	}
	// e.GET("/list", hdr.cacheEndpointHandler(hdr.MemoIndex))
	// e.POST("/", hdr.endpointHandler(hdr.MemoCreate))
	// e.DELETE("/:id", hdr.endpointHandler(hdr.MemoDelete))
	return hdr
}

func (h *MemoHandler) Connect() (database.Database, error) {
	redis, err := database.ConnectRedis()
	if err != nil {
		log.Printf("error: failed to Get memo data : %v\n", err)
		return nil, fmt.Errorf("failed to Get memo data: [%s]%w\n ", pkgName, err)
	}

	cached, err := redis.Exists()
	if err != nil {
		log.Printf("error: failed to Get cached data : %v\n", err)
		return nil, fmt.Errorf("failed to Get cached data: [%s]%w\n ", pkgName, err)
	}
	if cached {
		log.Printf("info: Found form Redis Memo cached data")
		h.HasCache = true
		return redis, nil
	}
	log.Printf("info: Not Found form Redis Memo cached data")
	return database.ConnectMySql()
}

func (h *MemoHandler) MemoIndex(c echo.Context) ([]byte, error) {

	memos, err := h.repo.Get()
	if err != nil {
		log.Printf("error: failed to Get memo data : %v\n", err)
		return nil, fmt.Errorf("failed to Get memo data: [%s]%w\n ", pkgName, err)
	}

	log.Printf("info: (%s)データ取得OK\n", pkgName)
	return memos, nil

}

// MemoCreate メモ作成
func (h *MemoHandler) MemoCreate(c echo.Context) ([]byte, error) {

	var (
		memo = &model.Memo{}
	)

	err := c.Bind(memo)
	if err != nil {
		log.Printf("error: 入力データに誤りがあります。:[%s] %v\n", pkgName, err)
		return nil, fmt.Errorf("failed to Bind request params :[%s] %v\n ", pkgName, err)
	}

	// バリデートを実行
	err = memo.Validate()
	if err != nil {
		log.Printf("error: バリデーションでエラーが発生しました。:[%s] %v\n", pkgName, err)
		return nil, fmt.Errorf("validation error:[%s] %w\n ", pkgName, err)
	}

	memoData, err := h.repo.Set(memo)
	if err != nil {
		log.Printf("error: データ挿入エラー :[%s] %v\n", pkgName, err)
		return nil, fmt.Errorf("failed to insert memo data :[%s] %w\n ", pkgName, err)
	}

	log.Printf(fmt.Sprintf("info: (%s)データ作成OK\n", pkgName))
	return memoData, nil
}

// MemoDelete メモ削除
func (h *MemoHandler) MemoDelete(c echo.Context) ([]byte, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Printf("error: データ型の変換エラー（int） :[%s] %v\n", pkgName, err)
		return nil, fmt.Errorf("failed to converted to type int :[%s] %w\n ", pkgName, err)
	}

	memoId, err := h.repo.DEL(id)
	if err != nil {
		log.Printf("error: データ削除エラー :[%s] %v\n", pkgName, err)
		return nil, fmt.Errorf("failed to delete memo data: [%s] %w\n ", pkgName, err)
	}

	log.Printf("info: データ削除OK[%s]\n", pkgName)
	return memoId, nil
}
