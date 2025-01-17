package distribution

import (
	"github.com/ALiuGuanyan/distributed-tasks-scheduling/microservices/entities"
	myconfig "github.com/ALiuGuanyan/distributed-tasks-scheduling/microservices/tasks-manager-worker-service/config"
	myutils "github.com/ALiuGuanyan/distributed-tasks-scheduling/microservices/utils"
	"sync"
	"time"
)

type scheduler interface {
	ScheduleLoop()
	TrySchedule() (scheduleAfter time.Duration)
	HandleTaskEvent(te *entities.TaskEvent)
	PushTaskEvent(te  *entities.TaskEvent)
	TryStartTask(tsp *entities.TaskSchedulePlan)
	PushTaskResultBack(ter *entities.TaskExecuteResult)
	HandleTaskResult(rst *entities.TaskExecuteResult)
}

//任务调度
type Scheduler struct {
	taskEventChan chan *entities.TaskEvent // etcd任务事件队列
	taskPlanTable map[string]*entities.TaskSchedulePlan // 任务调度计划表
	taskExcutingTable map[string]*entities.TaskExecuteInfo
	taskResultChan chan *entities.TaskExecuteResult
}

var (
	SgtScheduler *Scheduler
	onceScheduler sync.Once
)

// 初始化调度器单例
func InitScheduler() *Scheduler  {
	onceScheduler.Do(func() {
		SgtScheduler = &Scheduler{
			taskEventChan: make(chan *entities.TaskEvent, myconfig.ConfigSingleton.TaskEventCapacity),
			taskPlanTable: make(map[string]*entities.TaskSchedulePlan),
			taskExcutingTable: make(map[string]*entities.TaskExecuteInfo),
			taskResultChan: make(chan *entities.TaskExecuteResult, myconfig.ConfigSingleton.TaskEventCapacity),
		}
	})

	go SgtScheduler.ScheduleLoop()
	return SgtScheduler
}




// 调度协程
func (s *Scheduler) ScheduleLoop()  {
	var (
		taskEvent *entities.TaskEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		taskResult *entities.TaskExecuteResult
	)

	// 初始化一次
	scheduleAfter = s.TrySchedule()

	// 调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)
	// 定时任务entities.Task
	for {
		select {
		case taskEvent = <- s.taskEventChan: // 监听任务变化事件
			//对内存中维护的任务列表做增删改查
			s.HandleTaskEvent(taskEvent)

		case <-  scheduleTimer.C: // 最近的任务到期了

		case taskResult = <- s.taskResultChan: // 监听任务执行结果
			s.HandleTaskResult(taskResult)
		}
		// 调度一次任务
		scheduleAfter = s.TrySchedule()
		// 重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}
}

// 重新计算任务调度状态
func (s *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		tsp *entities.TaskSchedulePlan
		near *time.Time
	)

	// 如果任务表为空话，随便睡眠多久
	if len(s.taskPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}

	// 获取当前时间
	now := time.Now()

	// 1遍历所有任务
	for _, tsp = range s.taskPlanTable {

		if tsp.NextTime.Before(now) || tsp.NextTime.Equal(now) {
			// 尝试执行任务
			s.TryStartTask(tsp)
			tsp.NextTime = tsp.Expression.Next(now) // 更新下次执行时间
		}

		// 统计最近你一个要过期的任务时间
		if near == nil || tsp.NextTime.Before(*near) {
			near = &tsp.NextTime
		}
	}
	// 下次调度间隔 （最近要执行的任务调度时间-当前时间）
	if near != nil {
		scheduleAfter = (*near).Sub(now)
	}

	return
}

// 处理任务事件
func (s *Scheduler) HandleTaskEvent(te *entities.TaskEvent)  {
	var (
		tsp *entities.TaskSchedulePlan
		isExist bool
		err error
		tei *entities.TaskExecuteInfo
		isTaskExecuting bool
	)
	switch te.EventType {
	case entities.SAVE: // 保存任务事情
		if tsp, err = entities.BuildTaskSchedulePlan(te.Task);err != nil {
			return
		}
		s.taskPlanTable[te.Task.Name] = tsp
	case entities.DELETE: // 删除任务事件
		if tsp, isExist = s.taskPlanTable[te.Task.Name]; isExist {
			delete(s.taskPlanTable, te.Task.Name)
		}
	case entities.KILL:
		// 取消掉command执行
		tei, isTaskExecuting = s.taskExcutingTable[te.Task.Name]

		if  isTaskExecuting {

			tei.CancelFunc() // 触发command杀死shell子进程 任务得到退出
		}
	}
}

// 推送任务变化事件
func (s *Scheduler) PushTaskEvent(te  *entities.TaskEvent) {
	s.taskEventChan <- te
}

// 尝试执行任务
func (s *Scheduler) TryStartTask(tsp *entities.TaskSchedulePlan)  {
	// 调度和执行是两件事情
	var (
		tei *entities.TaskExecuteInfo
		isExecuting bool
	)
	// 执行的任务可能运行很久， 1分钟可能会调度60次，但只能执行一次, 防止并发

	// 如果任务正在执行 跳过此次调度
	if tei, isExecuting = s.taskExcutingTable[tsp.Task.Name]; isExecuting {
		return
	}

	// 构建执行状态信息
	tei = entities.BuildTaskExecuteInfo(tsp)
	// 保存执行状态
	s.taskExcutingTable[tsp.Task.Name] = tei

	// 执行任务
	SgtExecutor.ExecuteTask(tei)

}

// 回传任务执行结果
func (s *Scheduler) PushTaskResultBack(ter *entities.TaskExecuteResult)  {
	s.taskResultChan <- ter
}

// 处理任务结果
func (s *Scheduler) HandleTaskResult(rst *entities.TaskExecuteResult)  {
	// 删除执行状态
	delete(s.taskExcutingTable, rst.ExecuteInfo.Task.Name)

	var (
		taskLog *entities.TaskLog
	)
	// 生成执行日志
	if rst.Err != myutils.LockOccupiedError {
		taskLog = &entities.TaskLog{
			TaskName: rst.ExecuteInfo.Task.Name,
			Command: rst.ExecuteInfo.Task.Command,
			Output: string(rst.Output),
			PlanTime: rst.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: rst.ExecuteInfo.RealTime.UnixNano() / 1000/1000,
			StartTime: rst.StartTime.UnixNano() / 1000 /1000,
			EndTime: rst.EndTime.UnixNano() / 1000 / 1000,
		}

		if rst.Err != nil {
			taskLog.Error = rst.Err.Error()
		} else {
			taskLog.Error = ""
		}

		//存到MongoDB
		SgtSink.Append(taskLog)
	}
}