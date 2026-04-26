import { useState } from "react";
import type { FormEvent } from "react";
import type { NotificationChannel, NotificationProxy } from "../types";
import type {
  NotificationProxyFormState,
  TelegramFormState,
  TelegramPreviewState,
  WebhookFormState,
  WebhookPreviewState,
} from "../lib/app-types";
import { useI18n } from "../lib/i18n";

type NotificationsPageProps = {
  busy: boolean;
  notificationChannels: NotificationChannel[];
  notificationProxies: NotificationProxy[];
  notificationProxyForm: NotificationProxyFormState;
  notificationProxySaved: boolean;
  webhookForm: WebhookFormState;
  webhookSaved: boolean;
  webhookPreview: WebhookPreviewState | null;
  telegramForm: TelegramFormState;
  telegramSaved: boolean;
  telegramPreview: TelegramPreviewState | null;
  onNotificationProxyFormChange: (updater: (current: NotificationProxyFormState) => NotificationProxyFormState) => void;
  onSaveNotificationProxy: (event: FormEvent<HTMLFormElement>) => void;
  onEditNotificationProxy: (notificationProxy: NotificationProxy) => void;
  onCancelEditNotificationProxy: () => void;
  onDeleteNotificationProxy: (id: number) => void;
  onWebhookFormChange: (updater: (current: WebhookFormState) => WebhookFormState) => void;
  onTelegramFormChange: (updater: (current: TelegramFormState) => TelegramFormState) => void;
  onSaveWebhook: (event: FormEvent<HTMLFormElement>) => void;
  onTestWebhook: () => void;
  onTestTelegram: () => void;
  onSaveTelegram: (event: FormEvent<HTMLFormElement>) => void;
};

export default function NotificationsPage(props: NotificationsPageProps) {
  const { t } = useI18n();
  const [isWebhookConfigOpen, setWebhookConfigOpen] = useState(false);
  const notificationVariables = [
    { token: "{{machine_id}}", description: t("notificationsVariableMachineID") },
    { token: "{{machine_name}}", description: t("notificationsVariableMachineName") },
    { token: "{{machine_host}}", description: t("notificationsVariableMachineHost") },
    { token: "{{period_type}}", description: t("notificationsVariablePeriodType") },
    { token: "{{metric_type}}", description: t("notificationsVariableMetricType") },
    { token: "{{bucket_time}}", description: t("notificationsVariableBucketTime") },
    { token: "{{bucket_time_rfc3339}}", description: t("notificationsVariableBucketTimeRFC3339") },
    { token: "{{threshold_mb}}", description: t("notificationsVariableThresholdMB") },
    { token: "{{threshold_human_readable}}", description: t("notificationsVariableThresholdHumanReadable") },
    { token: "{{actual_mb}}", description: t("notificationsVariableActualMB") },
    { token: "{{actual_human_readable}}", description: t("notificationsVariableActualHumanReadable") },
    { token: "{{alert_key}}", description: t("notificationsVariableAlertKey") },
  ];
  const recommendedNotificationTemplate = `🚨 Bandwidth Limit Exceeded

🖥 {{machine_name}} ({{machine_host}})
📊 {{metric_type}} / {{period_type}}
📈 {{actual_human_readable}} / {{threshold_human_readable}}
🕒 {{bucket_time_rfc3339}}`;
  const notificationProxyLabel = (proxyID?: number) => {
    const notificationProxy = props.notificationProxies.find((proxy) => proxy.id === proxyID);
    if (notificationProxy) {
      return `${notificationProxy.name} (${notificationProxy.proxy_type})`;
    }

    return proxyID ? t("notificationsProxyMissing", { id: String(proxyID) }) : t("notificationsNoProxy");
  };

  return (
    <div className="page-stack">
      <section className="notification-status-grid" aria-label={t("notificationsChannelStatus")}>
        {props.notificationChannels.map((channel) => (
          <article className="card notification-status-card" key={channel.channel_type}>
            <div className="card-header">
              <div>
                <p className="section-kicker">{t("notificationsChannelStatus")}</p>
                <strong>{channel.channel_type}</strong>
              </div>
              <span className={`status-badge ${channel.enabled ? "ok" : "idle"}`}>
                {channel.enabled ? t("statusEnabled") : t("statusDisabled")}
              </span>
            </div>
            <p className="card-meta">
              {t("notificationsConfigured", { value: channel.configured ? t("statusConfiguredYes") : t("statusConfiguredNo") })}
            </p>
            {channel.url ? <p className="card-meta">{t("notificationsURL", { url: channel.url })}</p> : null}
            {channel.chat_id ? <p className="card-meta">{t("notificationsChatID", { value: channel.chat_id })}</p> : null}
            {channel.token_masked ? <p className="card-meta">{t("notificationsToken", { value: channel.token_masked })}</p> : null}
            <p className="card-meta">{t("notificationsProxy", { value: notificationProxyLabel(channel.proxy_id) })}</p>
          </article>
        ))}
      </section>

      <section className="panel section-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("notificationsProxyTitle")}</h3>
            </div>
            <p className="section-description">{t("notificationsProxyDescription")}</p>
          </div>
          <div className="action-row">
            {props.notificationProxyForm.id ? (
              <button className="secondary-button" onClick={props.onCancelEditNotificationProxy} type="button">
                {t("notificationsProxyCancelEdit")}
              </button>
            ) : null}
            <button
              className="primary-button"
              disabled={props.busy || props.notificationProxySaved}
              form="notification-proxy-form"
              type="submit"
            >
              {props.notificationProxyForm.id ? t("notificationsProxyUpdate") : t("notificationsProxyCreate")}
            </button>
          </div>
        </div>
        <form id="notification-proxy-form" className="form-grid notification-proxy-form-grid" onSubmit={props.onSaveNotificationProxy}>
          <label className="field">
            <span>{t("notificationsProxyName")}</span>
            <input
              value={props.notificationProxyForm.name}
              onChange={(event) => {
                props.onNotificationProxyFormChange((current) => ({ ...current, name: event.target.value }));
              }}
              placeholder={t("notificationsProxyNamePlaceholder")}
            />
          </label>
          <label className="field">
            <span>{t("notificationsProxyType")}</span>
            <select
              value={props.notificationProxyForm.proxyType}
              onChange={(event) => {
                props.onNotificationProxyFormChange((current) => ({
                  ...current,
                  proxyType: event.target.value as "http" | "socks",
                }));
              }}
            >
              <option value="http">HTTP</option>
              <option value="socks">SOCKS</option>
            </select>
          </label>
          <label className="field">
            <span>{t("notificationsProxyURL")}</span>
            <input
              value={props.notificationProxyForm.url}
              onChange={(event) => {
                props.onNotificationProxyFormChange((current) => ({ ...current, url: event.target.value }));
              }}
              placeholder={props.notificationProxyForm.proxyType === "socks" ? "socks5://127.0.0.1:1080" : "http://127.0.0.1:7890"}
            />
          </label>
        </form>
        <div className="table-wrapper notification-proxy-table-wrapper">
          <table className="notification-proxy-table">
            <thead>
              <tr>
                <th>{t("notificationsProxyName")}</th>
                <th>{t("notificationsProxyType")}</th>
                <th>{t("notificationsProxyURL")}</th>
                <th>{t("notificationsProxyActions")}</th>
              </tr>
            </thead>
            <tbody>
              {props.notificationProxies.length > 0 ? (
                props.notificationProxies.map((notificationProxy) => (
                  <tr key={notificationProxy.id}>
                    <td>{notificationProxy.name}</td>
                    <td>{notificationProxy.proxy_type.toUpperCase()}</td>
                    <td className="table-text-muted">{notificationProxy.url}</td>
                    <td>
                      <div className="action-row">
                        <button className="secondary-button" onClick={() => props.onEditNotificationProxy(notificationProxy)} type="button">
                          {t("notificationsProxyEdit")}
                        </button>
                        <button className="danger-button" onClick={() => props.onDeleteNotificationProxy(notificationProxy.id)} type="button">
                          {t("notificationsProxyDelete")}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td className="table-text-muted" colSpan={4}>
                    {t("notificationsProxyEmpty")}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel section-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("notificationsWebhookTitle")}</h3>
            </div>
            <p className="section-description">{t("notificationsPageDescription")}</p>
          </div>
          <div className="action-row">
            <button className="secondary-button" onClick={() => setWebhookConfigOpen((current) => !current)} type="button">
              {isWebhookConfigOpen ? t("notificationsCollapseConfig") : t("notificationsExpandConfig")}
            </button>
            <button className="secondary-button" disabled={props.busy} onClick={props.onTestWebhook} type="button">
              {t("notificationsTestWebhook")}
            </button>
            <button
              className="primary-button"
              disabled={props.busy || props.webhookSaved}
              form="webhook-form"
              type="submit"
            >
              {t("notificationsSaveWebhook")}
            </button>
          </div>
        </div>
        <form id="webhook-form" className="form-grid webhook-form-grid" onSubmit={props.onSaveWebhook}>
          <label className="field checkbox-field full-width">
            <input
              checked={props.webhookForm.enabled}
              onChange={(event) => {
                props.onWebhookFormChange((current) => ({ ...current, enabled: event.target.checked }));
              }}
              type="checkbox"
            />
            <span>{t("notificationsEnableWebhook")}</span>
          </label>

          {isWebhookConfigOpen ? (
            <>
              <label className="field">
                <span>{t("notificationsMethod")}</span>
                <select
                  value={props.webhookForm.method}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({
                      ...current,
                      method: event.target.value as "GET" | "POST",
                    }));
                  }}
                >
                  <option value="POST">POST</option>
                  <option value="GET">GET</option>
                </select>
              </label>
              <label className="field">
                <span>URL</span>
                <input
                  value={props.webhookForm.url}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, url: event.target.value }));
                  }}
                  placeholder="https://example.com/hook"
                />
              </label>
              <label className="field">
                <span>{t("notificationsProxySelect")}</span>
                <select
                  value={props.webhookForm.proxyID}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, proxyID: event.target.value }));
                  }}
                >
                  <option value="">{t("notificationsNoProxy")}</option>
                  {props.notificationProxies.map((notificationProxy) => (
                    <option key={notificationProxy.id} value={String(notificationProxy.id)}>
                      {notificationProxyLabel(notificationProxy.id)}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field full-width">
                <span>{t("notificationsHeaders")}</span>
                <textarea
                  rows={4}
                  value={props.webhookForm.headersText}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, headersText: event.target.value }));
                  }}
                  placeholder={`{
  "Authorization": "Bearer {{alert_key}}",
  "X-Metric": "{{metric_type}}",
  "X-Machine-Name": "{{machine_name}}"
}`}
                />
              </label>
              <label className="field full-width">
                <span>{t("notificationsBody")}</span>
                <textarea
                  rows={7}
                  value={props.webhookForm.bodyText}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, bodyText: event.target.value }));
                  }}
                  placeholder={`{
  "machine_id": "{{machine_id}}",
  "machine_name": "{{machine_name}}",
  "machine_host": "{{machine_host}}",
  "period_type": "{{period_type}}",
  "metric_type": "{{metric_type}}",
  "bucket_time": "{{bucket_time}}",
  "threshold_mb": "{{threshold_mb}}",
  "threshold_human_readable": "{{threshold_human_readable}}",
  "actual_mb": "{{actual_mb}}",
  "actual_human_readable": "{{actual_human_readable}}",
  "alert_key": "{{alert_key}}"
}`}
                />
              </label>
              <div className="card">
                <div className="card-header">
                  <strong>{t("notificationsVariablesTitle")}</strong>
                </div>
                <p className="card-meta">
                  {t("notificationsVariablesDesc")}
                </p>
                <dl className="variable-description-list">
                  {notificationVariables.map((variable) => (
                    <div className="variable-description-item" key={variable.token}>
                      <dt><code>{variable.token}</code></dt>
                      <dd>{variable.description}</dd>
                    </div>
                  ))}
                </dl>
                <p className="card-meta notification-template-title">{t("notificationsRecommendedTemplate")}</p>
                <pre className="code-block">{recommendedNotificationTemplate}</pre>
              </div>
              {props.webhookPreview ? (
                <div className="card">
                  <div className="card-header">
                    <strong>{t("notificationsPreviewTitle")}</strong>
                  </div>
                  <p className="card-meta">{t("notificationsPreviewURL")}</p>
                  <pre className="code-block">{props.webhookPreview.url || "-"}</pre>
                  <p className="card-meta">{t("notificationsPreviewHeaders")}</p>
                  <pre className="code-block">{props.webhookPreview.headersText || "{}"}</pre>
                  <p className="card-meta">{t("notificationsPreviewBody")}</p>
                  <pre className="code-block">{props.webhookPreview.bodyText || "-"}</pre>
                </div>
              ) : null}
            </>
          ) : (
            <div className="card webhook-collapsed-note full-width">
              <p className="card-meta">{t("notificationsWebhookCollapsed")}</p>
              {props.webhookForm.url ? <p className="card-meta">{t("notificationsURL", { url: props.webhookForm.url })}</p> : null}
              <p className="card-meta">{t("notificationsProxy", { value: notificationProxyLabel(Number(props.webhookForm.proxyID) || undefined) })}</p>
            </div>
          )}
        </form>
      </section>

      <section className="panel section-panel telegram-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("notificationsTelegramTitle")}</h3>
            </div>
            <p className="section-description">{t("notificationsTelegramDescription")}</p>
          </div>
          <div className="action-row">
            <button className="secondary-button" disabled={props.busy} onClick={props.onTestTelegram} type="button">
              {t("notificationsTestTelegram")}
            </button>
            <button
              className="primary-button"
              disabled={props.busy || props.telegramSaved}
              form="telegram-form"
              type="submit"
            >
              {t("notificationsSaveTelegram")}
            </button>
          </div>
        </div>
        <form id="telegram-form" className="form-grid telegram-form-grid" onSubmit={props.onSaveTelegram}>
          <label className="field checkbox-field full-width">
            <input
              checked={props.telegramForm.enabled}
              onChange={(event) => {
                props.onTelegramFormChange((current) => ({ ...current, enabled: event.target.checked }));
              }}
              type="checkbox"
            />
            <span>{t("notificationsEnableTelegram")}</span>
          </label>

          {props.telegramForm.enabled ? (
            <>
              <label className="field">
                <span>Bot Token</span>
                <input
                  value={props.telegramForm.botToken}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, botToken: event.target.value }));
                  }}
                  placeholder={t("notificationsBotTokenPlaceholder")}
                />
              </label>
              <label className="field">
                <span>Chat ID</span>
                <input
                  value={props.telegramForm.chatID}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, chatID: event.target.value }));
                  }}
                />
              </label>
              <label className="field">
                <span>{t("notificationsProxySelect")}</span>
                <select
                  value={props.telegramForm.proxyID}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, proxyID: event.target.value }));
                  }}
                >
                  <option value="">{t("notificationsNoProxy")}</option>
                  {props.notificationProxies.map((notificationProxy) => (
                    <option key={notificationProxy.id} value={String(notificationProxy.id)}>
                      {notificationProxyLabel(notificationProxy.id)}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field full-width">
                <span>{t("notificationsTelegramMessage")}</span>
                <textarea
                  rows={5}
                  value={props.telegramForm.messageText}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, messageText: event.target.value }));
                  }}
                  placeholder="machine={{machine_name}} actual={{actual_human_readable}}"
                />
              </label>
              <div className="card">
                <div className="card-header">
                  <strong>{t("notificationsVariablesTitle")}</strong>
                </div>
                <p className="card-meta">
                  {t("notificationsTelegramVariablesDesc")}
                </p>
                <dl className="variable-description-list">
                  {notificationVariables.map((variable) => (
                    <div className="variable-description-item" key={variable.token}>
                      <dt><code>{variable.token}</code></dt>
                      <dd>{variable.description}</dd>
                    </div>
                  ))}
                </dl>
                <p className="card-meta notification-template-title">{t("notificationsRecommendedTemplate")}</p>
                <pre className="code-block">{recommendedNotificationTemplate}</pre>
              </div>
              {props.telegramPreview ? (
                <div className="card">
                  <div className="card-header">
                    <strong>{t("notificationsPreviewTitle")}</strong>
                  </div>
                  <p className="card-meta">{t("notificationsPreviewMessage")}</p>
                  <pre className="code-block">{props.telegramPreview.messageText || "-"}</pre>
                </div>
              ) : null}
            </>
          ) : (
            <div className="card telegram-collapsed-note full-width">
              <p className="card-meta">{t("notificationsTelegramCollapsed")}</p>
              <p className="card-meta">{t("notificationsProxy", { value: notificationProxyLabel(Number(props.telegramForm.proxyID) || undefined) })}</p>
            </div>
          )}
        </form>
      </section>
    </div>
  );
}
