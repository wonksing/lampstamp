# 램포트 타임스탬프(Lamport Timestamp)

## What is it?
램포트 타임스탬프는 인과적 순서를 알려주는 논리적인 시계이다. 예를 들면, "밥을 먹고(A) 이를 닦는다.(B)"라는 2가지 이벤트 A와 B가 있다. 이를 닦은 것은 밥을 먹은 후에 일어났다는 인과 관계를 가지고 있다. 램포트 타임스탬프는 이런 순서를 알려주는 역할을 하는 것이다. 

사용자의 모든 행위를 기록하는 커머스앱이 있다고 하자. 한 사용자가 아이폰14를 검색(A)하고 프로 맥스 사진을 누르고 상세화면(B)으로 들어가서 주문(C) 버튼을 눌렀다. 사용자는 "검색 > 상세화면 진입 > 주문" 행위를 순서대로 진행했다. 그런데, 여러 서버로 구성된 분산 환경에서 이런 이벤트를 처리할 때 "주문 > 검색 > 상세화면 진입" 순서로 저장될 수 있다. API 서비스와 저장소간의 네트워크 지연이나 부하로 인해 한 서버가 다른 서버보다 먼저 이벤트를 저장할 가능성이 있기 때문이다.(그렇다..나는 아이폰14을 갖고 싶다!!)

램포트 타임스탬프는 단순한 시퀀스다. 데이터를 보내거나 쓸때 노드(서버나 클라이언트)가 갖고 있는 타임스탬프와 전달받은 타임스탬프 중 최댓값에 1을 증가시킨다. 이 새로운 값을 노드 자신의 값으로 대치하고 데이터와 함께 send/write 한다. 자세한 내용은 여기서(https://martinfowler.com/articles/patterns-of-distributed-systems/lamport-clock.html) 확인 가능하다.

## What about it?
어느날 서버와 클라이언트의 값이 다른것이 발견됐다. 서버는 클라이언트가 최초로 저장했던 값을 가지고 있고 클라이언트는 최초 저장한 값을 변경한 값을 가지고 있는 것이다. 그리고 지는 정상적으로 보내고 서버가 응답했다는 증거도 있다.

이 시스템은 다음과 같이 구성했다. 클라이언트는 로컬 디비를 가지고 있으며 데이터는 여기에 먼저 저장된다. 별도의 워커 쓰레드가 변경된 데이터를 찾아서 서버로 보낸다. 한번 저장한 데이터는 사용자 행위에 따라서 값이 바뀌기도 한다. 서버는 여러개의 API 서비스로 구성되어 있고 단일 리더 노드(DB)에만 쓰기를 한다. 

이런 결과를 만들 수 있는 가설, 상상, 테스트를 하다가 램포트 시계를 사용해 보기로 했다. 이 문제를 해결하기에 적절한 것으로 보인다. 그럼 이런 문제를 만들고 해결해 보도록 하자.

## Problem
### 문제 재현
클라이언트, API 서버 그리고 저장소를 만든다. 2개의 클라이언트가 2개의 API 서버로 메시지를 보내고 1개의 스토리지에 이 메시지를 저장한다.
 
메시지는 ID, ClientID, Text, 3개의 필드를 가진 구조체다. 동일한 메시지를 2개의 클라이언트가 서로 변경해야 하는 상황을 만들어야 하기 때문에 메시지의 ID는 동일한 것을 사용하고 Text는 서로 다르게 한다. 저장소의 메시지는 메시지의 ID를 키로 저장한다.
```go
type Message struct {
	ID       string 
	ClientID string 
	Text     string // 변경할 텍스트
}
```

`svr-1`에는 모든 메시지를 저장할 때 0.2초의 지연을 줘서 `svr-2`보다 늦게 처리되도록 한다. `client-1`이 `client-2`보다 먼저 보내야 하는 상황을 만들어야 하기 때문에 `client-1`이 메시지를 전송하고 0.1초 후에 `client-2`가 메시지를 전송하도록 한다. 

1. `client-1`이 `svr-1`으로 `foo`를 보낸다.
2. 0.1초 sleep 
3. `client-2`가 `svr-2`로 `bar`를 보낸다.
4. 저장소에 저장되어 있는 메시지를 확인한다.

`client-1`이 `foo`를 보낸 후에 `client-2`가 `bar`를 보냈기 때문에 저장소에는 `bar`가 저장되어 있는게 정상이지만 `svr-1`의 지연으로 인해 `foo`가 저장되어 있는것이 확인된다.(`Message in storage: foo`)
```
10:57:48 client-1 sends 'foo'
10:57:48 client-2 sends 'bar'
10:57:48 	svr-2 responds 'bar'
10:57:48 		client-2 received 'bar'
10:57:48 	svr-1 responds 'foo'
10:57:48 		client-1 received 'foo'
10:57:53 Message in storage: foo
```

### 램포트 타임스탬프 적용
램포트 타임스탬프를 적용해서 메시지가 어떤 순서로 처리되는지 확인해 보자. 메시지 구조체에 `Version`이라는 필드를 추가하여 타임스탬프를 기록할 수 있게 한다.
```go
type Message struct {
	ID       string
	ClientID string
	Text     string

	Version int64 // 버전(타임스탬프)
}
```

모든 노드(클라이언트, API 서버)는 메시지 ID 단위로 타임스탬프를 가지고 있고 send/write 할 때 자신이 가지고 있는 타임스탬프와 메시지의 타임스탬프 둘 중 최댓값에 1을 더한 값을 자신과 메시지의 타임스탬프로 대치한다.

다음은 타임스탬프를 함께 기록한 로그다. 타임스탬프를 통해 순서를 알 수는 있지만 각 메시지가 서로 다른 서버에서 처리되었기 때문에 정확히 알 수 없고 결과 역시 앞서 봤던 것과 같이 `foo`가 저장소에 있는 것이 확인된다.
```
11:15:55 client-1 sends 'foo(version: 1)'
11:15:55 client-2 sends 'bar(version: 1)'
11:15:55 	svr-2 responds 'bar(version: 2)'
11:15:55 		client-2 received 'bar(version: 2)'
11:15:55 	svr-1 responds 'foo(version: 2)'
11:15:55 		client-1 received 'foo(version: 2)'
11:16:00 message in storage: foo(version: 2)
```

### 타임스탬프 검사
API 서버에서 메시지를 저장소에 저장하기 전에 타임스탬프를 검사하여 가장 최근의 것만 저장하도록 다음과 같은 체크 로직을 추가해 보자.
```go
    ...
    ...
	s.storage.Begin()
	defer s.storage.End()

	storageMsgVersion := s.storage.ReadVersion(msg.ID)
	if storageMsgVersion >= msg.Version {
		msg.Failed = true
		log.Printf("\tERROR %s: msg version is not higher than storage's, '%s'\n", s.ID, msg.String())
		clientChannel, _ := s.clientChannels.Load(msg.ClientID)
		clientChannel.(chan Message) <- msg
		return
	}

	s.storage.Store(msg)
    ...
    ...
```

실행 후 로그를 확인해 보자. 이제, 더 나중에 전송한 `client-2`의 `bar`가 저장된 것을 확인할 수 있다.
```
11:32:53 client-1 sends 'foo'
11:32:54 client-2 sends 'bar'
11:32:54 	svr-2 responds 'bar'
11:32:54 		client-2 received 'bar'
11:32:54 	ERROR svr-1: msg version is not higher than storage's
11:32:54 		ERROR: client-1 received 'foo'
11:32:59 Message in storage: bar
```

## Consideration
1. 만약 쓰기 저장소가 2개 이상이면 램포트 타임스탬프로 전체 순서를 판단할 수 없다.
2. 저장소의 메시지 버전을 확인하는 비용이 추가된다.(확인을 하지 않는 로직을 만들어야 함)
3. 메시지별로 타임스탬프를 관리하면 API 서버의 메모리가 지속적으로 증가하게 된다.
   - 로컬캐시로 오래된 것을 지우거나
   - Redis나 Memcached로 중앙 관리하거나

## Conclusion
이런 상황은 내가 인지하지 못하는 것일 뿐 더 자주 발생하고 있을 것으로 예상한다. 메시지 쓰기(write)에 대한 것만 테스트 했지만 읽기(read)에도 비슷한 방법을 사용할 수 있을 것으로 생각된다. 