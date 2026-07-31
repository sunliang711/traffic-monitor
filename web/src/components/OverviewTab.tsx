import type { CollectNowResponse, Machine, NotificationChannel, SSHKey } from "../types";
import { useI18n } from "../lib/i18n";
import { formatStatusText } from "../lib/app-utils";

export type OverviewTabKey = "sshKeys" | "machines" | "notifications" | "samples" | "alerts";

type OverviewTabProps = {
  sshKeys: SSHKey[];
  machines: Machine[];
  notificationChannels: NotificationChannel[];
  samplesTotal: number;
  alertsTotal: number;
  collectResults: CollectNowResponse["results"];
  readOnly?: boolean;
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
  const configuredChannels = props.notificationChannels.filter((channel) => channel.configured).length;
  const latestStatus = props.collectResults.length ? formatStatusText(props.collectResults[0].status, language) : t("statusNotRun");

  return (
    <div className="overview-layout">
      <section className="summary-strip">
        <SummaryTile label={t("overviewEnabledMachines")} value={String(enabledMachines)} tone="teal" />
        <SummaryTile label={t("overviewConfiguredNotifications")} value={String(configuredChannels)} tone="amber" />
        <SummaryTile label={t("overviewLatestStatus")} value={latestStatus} tone="slate" />
      </section>

      <div className="grid overview-grid">
        {props.readOnly ? null : (
          <StatCard
            label={t("overviewSSHKeysLabel")}
            value={String(props.sshKeys.length)}
            help={t("overviewSSHKeysHelp")}
            onClick={() => props.onNavigate("sshKeys")}
          />
        )}
        <StatCard
          label={t("overviewMachinesLabel")}
          value={String(props.machines.length)}
          help={t("overviewMachinesHelp", { count: enabledMachines })}
          onClick={() => props.onNavigate("machines")}
        />
        {props.readOnly ? null : (
          <StatCard
            label={t("overviewNotificationsLabel")}
            value={String(enabledChannels)}
            help={t("overviewNotificationsHelp")}
            onClick={() => props.onNavigate("notifications")}
          />
        )}
        <StatCard
          label={t("overviewSamplesLabel")}
          value={String(props.samplesTotal)}
          help={t("overviewSamplesHelp")}
          onClick={() => props.onNavigate("samples")}
        />
        <StatCard
          label={t("overviewAlertsLabel")}
          value={String(props.alertsTotal)}
          help={t("overviewAlertsHelp")}
          onClick={() => props.onNavigate("alerts")}
        />
        <StatCard
          label={t("overviewCollectLabel")}
          value={latestStatus}
          help={t("overviewCollectHelp")}
        />
      </div>

      <section className="overview-storyboard">
        <article className="story-card">
          <p className="eyebrow">{t("overviewSummaryTitle")}</p>
          <h4>{t("overviewOperationsTitle")}</h4>
          <p>{t("overviewOperationsDescription")}</p>
        </article>
        <article className="story-card story-card-highlight">
          <div className="story-card-grid">
            <div>
              <span className="story-kicker">{t("tabMachines")}</span>
              <strong>{enabledMachines}</strong>
            </div>
            <div>
              <span className="story-kicker">{t("tabNotifications")}</span>
              <strong>{enabledChannels}</strong>
            </div>
            <div>
              <span className="story-kicker">{t("tabSamples")}</span>
              <strong>{props.samplesTotal}</strong>
            </div>
            <div>
              <span className="story-kicker">{t("tabAlerts")}</span>
              <strong>{props.alertsTotal}</strong>
            </div>
          </div>
        </article>
      </section>
    </div>
  );
}

function StatCard(props: StatCardProps) {
  const { t } = useI18n();
  const content = (
    <>
      <p className="stat-card-label">{props.label}</p>
      <h3>{props.value}</h3>
      <p className="card-meta">{props.help}</p>
      {props.onClick ? <span className="stat-card-link">{t("openAction")}</span> : null}
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

function SummaryTile(props: { label: string; value: string; tone: "teal" | "amber" | "slate" }) {
  return (
    <article className={`summary-tile ${props.tone} compact`}>
      <span>{props.label}</span>
      <strong>{props.value}</strong>
    </article>
  );
}
