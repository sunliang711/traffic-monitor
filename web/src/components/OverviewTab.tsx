import type {
  AlertItem,
  CollectNowResponse,
  Machine,
  NotificationChannel,
  SSHKey,
  TrafficSample,
} from "../types";

export type OverviewTabKey = "sshKeys" | "machines" | "notifications" | "samples" | "alerts";

type OverviewTabProps = {
  sshKeys: SSHKey[];
  machines: Machine[];
  notificationChannels: NotificationChannel[];
  samples: TrafficSample[];
  alerts: AlertItem[];
  collectResults: CollectNowResponse["results"];
  onNavigate: (tab: OverviewTabKey) => void;
};

type StatCardProps = {
  label: string;
  value: string;
  help: string;
  onClick?: () => void;
};

export default function OverviewTab(props: OverviewTabProps) {
  const enabledMachines = props.machines.filter((machine) => machine.collect_enabled).length;
  const enabledChannels = props.notificationChannels.filter((channel) => channel.enabled).length;

  return (
    <div className="grid overview-grid">
      <StatCard
        label="SSH Key"
        value={String(props.sshKeys.length)}
        help="当前可用的登录密钥数量"
        onClick={() => props.onNavigate("sshKeys")}
      />
      <StatCard
        label="机器总数"
        value={String(props.machines.length)}
        help={`启用采集 ${enabledMachines} 台`}
        onClick={() => props.onNavigate("machines")}
      />
      <StatCard
        label="通知渠道"
        value={String(enabledChannels)}
        help="已启用的通知渠道数量"
        onClick={() => props.onNavigate("notifications")}
      />
      <StatCard
        label="最近样本"
        value={String(props.samples.length)}
        help="当前查询到的样本条数"
        onClick={() => props.onNavigate("samples")}
      />
      <StatCard
        label="告警总数"
        value={String(props.alerts.length)}
        help="当前查询到的告警条数"
        onClick={() => props.onNavigate("alerts")}
      />
      <StatCard
        label="最近采集执行"
        value={props.collectResults.length ? props.collectResults[0].status : "未执行"}
        help="手动采集的最近一次结果"
      />
    </div>
  );
}

function StatCard(props: StatCardProps) {
  const content = (
    <>
      <p className="muted">{props.label}</p>
      <h3>{props.value}</h3>
      <p className="card-meta">{props.help}</p>
    </>
  );

  if (props.onClick) {
    return (
      <button className="panel stat-card" onClick={props.onClick} type="button">
        {content}
      </button>
    );
  }

  return <section className="panel stat-card">{content}</section>;
}
