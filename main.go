package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	queryCorrectTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "query_correct_Total",
		Help: "请求正确的个数",
	})
	queryDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "query_durations_seconds",
			Help:       "query tp 单位为ms ",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001, 0.9999: 0.00001},
		},
		nil,
	)

	conf *Conf
)

type Conf struct {
	Domain             string
	QueryRangeUri      string
	QueryRanges        []string
	Qps                int64
	ExecutionTime      int
	QueryRangeDuration int64
	Port               int
}

func init() {
	prometheus.MustRegister(queryCorrectTotal)
	prometheus.MustRegister(queryDurations)
}

func main() {
	conf = &Conf{}
	a := kingpin.New(filepath.Base(os.Args[0]), "prometheus 查询语句压力测试工具")
	a.Flag("prometheus.domain", "Prometheus addr").StringVar(&conf.Domain)
	a.Flag("prometheus.query_range_uri", "Prometheus addr").StringVar(&conf.QueryRangeUri)
	a.Flag("prometheus.query_ranges", "Prometheus 查询语句列表").StringsVar(&conf.QueryRanges)
	a.Flag("prometheus.qps", "Qps数").Int64Var(&conf.Qps)
	a.Flag("prometheus.execution_time", "压测执行时间，单位为秒").IntVar(&conf.ExecutionTime)
	a.Flag("prometheus.query_range_duration", "查询区间时间跨度，单位为秒").Int64Var(&conf.QueryRangeDuration)
	a.Flag("prometheus.http_port", "端口").Default("8080").IntVar(&conf.Port)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "命令行解析错误"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*time.Duration(conf.ExecutionTime))

	phs := NewPHS(fmt.Sprintf(":%d", conf.Port), nil)
	go phs.Start()
	StressTest(ctx)
	defer func() {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", conf.Port))
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
		if resp != nil {
			resBytes, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(resBytes))
		}
		phs.Stop()
	}()
}

func StressTest(ctx context.Context) {
	fmt.Printf("test url : %s%s", conf.Domain, conf.QueryRangeUri)
	fmt.Println()
	querySlice, err := getQuerys()
	if err != nil {
		return
	}
	querySliceLen := len(querySlice)
	if querySliceLen < 1 {
		fmt.Fprintln(os.Stderr, "初始化查询语句为空")
		return
	}

	fmt.Println("-----------------------start-----------------------")
	uri := fmt.Sprintf("%s%s", conf.Domain, conf.QueryRangeUri)
	tt := time.NewTicker(time.Second)
	defer func() {
		fmt.Println("-----------------------end-----------------------")
		tt.Stop()
	}()
	wg := &sync.WaitGroup{}
	testQueryOne := func(query string) {
		queryEscape := url.QueryEscape(query)
		wg.Add(1)
		rand.Seed(time.Now().UnixNano())
		timeSN := time.Now().UnixNano()
		defer func() {
			wg.Done()
			timeSNE := time.Now().UnixNano()
			takeTime := float64(timeSNE-timeSN) / 1000000
			queryDurations.WithLabelValues().Observe(takeTime)
		}()

		timeS := time.Now().Unix()
		url := fmt.Sprintf(`%s?query=%s&start=%d&end=%d&step=60`, uri, queryEscape, timeS-conf.QueryRangeDuration, timeS)
		resp, _ := http.Get(url)
		if resp != nil && resp.StatusCode == 200 {
			queryCorrectTotal.Inc()
		}
	}
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			fmt.Println("wg wait end ")
			time.Sleep(2)
			return
		case <-tt.C:
			for i := 0; i < int(conf.Qps); i++ {
				rand.Seed(time.Now().UnixNano())
				index := rand.Intn(querySliceLen)
				query := querySlice[index]
				go testQueryOne(query)
			}
		}
	}
}

func getQuerys() ([]string, error) {
	uri := fmt.Sprintf("%s%s", conf.Domain, conf.QueryRangeUri)
	timeS := time.Now().Unix()
	querySlice := make([]string, 0, 1024)

	qFunc := func(query string) error {
		queryEscape := url.QueryEscape(query)
		url := fmt.Sprintf(`%s?query=%s&start=%d&end=%d&step=60`, uri, queryEscape, timeS-300, timeS)
		resp, err := http.Get(url)
		if err != nil {
			return errors.New("从prometheus 获取初始化查询语句时间序列报错")
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		res, err := simplejson.NewJson(body)
		if err != nil {
			return err
		}
		rows, err := res.Get("data").Get("result").Array()
		for _, row := range rows {
			if eachMap, ok := row.(map[string]interface{}); ok {
				if metric, ok := eachMap["metric"].(map[string]interface{}); ok {
					tagKVs := make([]string, 0, len(metric))
					delete(metric, "__name__")
					for k, v := range metric {
						tagKVs = append(tagKVs, fmt.Sprintf(`%s="%s"`, k, v))
					}
					if len(tagKVs) > 0 {
						tags := strings.Join(tagKVs, ",")
						tag := strings.Replace(query, "{", fmt.Sprintf("{%s,", tags), -1)
						querySlice = append(querySlice, tag)
					}
				}
			}
		}
		return nil
	}
	for _, query := range conf.QueryRanges {
		if err := qFunc(query); err != nil {
			return nil, err
		}
	}
	return querySlice, nil
}
