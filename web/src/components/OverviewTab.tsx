import type {
  AlertItem,
  CollectNowResponse,
  Machine,
  NotificationChannel,
  SSHKey,
  TrafficSample,
} from "../types";
import { useI18n } from "../lib/i18n";
import { formatStatusText } from "../lib/app-utils";

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
  const { language, t } = useI18n();
  const enabledMachines = props.machines.filter((machine) => machine.collect_enabled).length;
  const enabledChannels = props.notificationChannels.filter((channel) => channel.enabled).length;

  return (
    <div className="grid overview-grid">
      <StatCard
        label={t("overviewSSHKeysLabel")}
        value={String(props.sshKeys.length)}
        help={t("overviewSSHKeysHelp")}
        onClick={() => props.onNavigate("sshKeys")}
      />
      <StatCard
        label={t("overviewMachinesLabel")}
        value={String(props.machines.length)}
        help={t("overviewMachinesHelp", { count: enabledMachines })}
        onClick={() => props.onNavigate("machines")}
      />
      <StatCard
        label={t("overviewNotificationsLabel")}
        value={String(enabledChannels)}
        help={t("overviewNotificationsHelp")}
        onClick={() => props.onNavigate("notifications")}
      />
      <StatCard
        label={t("overviewSamplesLabel")}
        value={String(props.samples.length)}
        help={t("overviewSamplesHelp")}
        onClick={() => props.onNavigate("samples")}
      />
      <StatCard
        label={t("overviewAlertsLabel")}
        value={String(props.alerts.length)}
        help={t("overviewAlertsHelp")}
        onClick={() => props.onNavigate("alerts")}
      />
      <StatCard
        label={t("overviewCollectLabel")}
        value={props.collectResults.length ? formatStatusText(props.collectResults[0].status, language) : t("statusNotRun")}
        help={t("overviewCollectHelp")}
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
