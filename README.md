# prom_query_stress_testing
prometheus 压力测试工具

```
查询语句要包含{},实例如下：
--prometheus.domain=http://autocmp-ds-one.corpautohome.com --prometheus.query_range_uri=/api/v1/query_range  --prometheus.query_ranges="count({__name__=~\"java_lang_garbagecollector.*|java_lang_memorypool_valid|java_lang_classloading.*\",_job=\"kafka-monitor\"}) by (__name__, cluster)" --prometheus.qps=100 --prometheus.execution_time=20 --prometheus.query_range_duration=21600 --prometheus.http_port=9090
```


```
# HELP query_correct_Total 请求正确的个数
# TYPE query_correct_Total counter
query_correct_Total 199299
# HELP query_durations_seconds query tp 
# TYPE query_durations_seconds summary
query_durations_seconds{quantile="0.5"} 35.445173
query_durations_seconds{quantile="0.9"} 72.946308
query_durations_seconds{quantile="0.95"} 60017.347092
query_durations_seconds{quantile="0.99"} 120014.107999
query_durations_seconds{quantile="0.9999"} 120085.28974
query_durations_seconds_sum 9.975450861595895e+08
query_durations_seconds_count 203830
```


```
prometheus 查询语句压力测试工具

Flags:
  --help                       Show context-sensitive help (also try --help-long and --help-man).
  --prometheus.domain=PROMETHEUS.DOMAIN  
                               Prometheus addr
  --prometheus.query_range_uri=PROMETHEUS.QUERY_RANGE_URI  
                               Prometheus addr
  --prometheus.query_ranges=PROMETHEUS.QUERY_RANGES ...  
                               Prometheus 查询语句列表
  --prometheus.qps=PROMETHEUS.QPS  
                               Qps数
  --prometheus.execution_time=PROMETHEUS.EXECUTION_TIME  
                               压测执行时间，单位为秒
  --prometheus.query_range_duration=PROMETHEUS.QUERY_RANGE_DURATION  
                               查询区间时间跨度，单位为秒
  --prometheus.http_port=8080  端口

```