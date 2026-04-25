import type { FormEvent } from "react";
import type { NotificationChannel } from "../types";
import type { TelegramFormState, WebhookFormState, WebhookPreviewState } from "../lib/app-types";

type NotificationsPageProps = {
  busy: boolean;
  notificationChannels: NotificationChannel[];
  webhookForm: WebhookFormState;
  webhookSaved: boolean;
  webhookPreview: WebhookPreviewState | null;
  telegramForm: TelegramFormState;
  telegramSaved: boolean;
  onWebhookFormChange: (updater: (current: WebhookFormState) => WebhookFormState) => void;
  onTelegramFormChange: (updater: (current: TelegramFormState) => TelegramFormState) => void;
  onSaveWebhook: (event: FormEvent<HTMLFormElement>) => void;
  onTestWebhook: () => void;
  onSaveTelegram: (event: FormEvent<HTMLFormElement>) => void;
};

export default function NotificationsPage(props: NotificationsPageProps) {
  return (
    <div className="grid two-columns">
      <section className="panel">
        <h3 className="panel-title">Webhook 通知</h3>
        <form className="form-grid" onSubmit={props.onSaveWebhook}>
          <label className="field checkbox-field">
            <input
              checked={props.webhookForm.enabled}
              onChange={(event) => {
                props.onWebhookFormChange((current) => ({ ...current, enabled: event.target.checked }));
              }}
              type="checkbox"
            />
            <span>启用 Webhook</span>
          </label>
          <label className="field">
            <span>请求方式</span>
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
          <label className="field full-width">
            <span>Headers(JSON 模板)</span>
            <textarea
              rows={5}
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
            <span>Body 模板</span>
            <textarea
              rows={8}
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
  "actual_mb": "{{actual_mb}}",
  "alert_key": "{{alert_key}}"
}`}
            />
          </label>
          <div className="card">
            <div className="card-header">
              <strong>可用变量</strong>
            </div>
            <p className="card-meta">
              URL、Headers、Body 都支持以下变量模板：
              <code> {"{{machine_id}}"}</code>
              <code> {"{{machine_name}}"}</code>
              <code> {"{{machine_host}}"}</code>
              <code> {"{{period_type}}"}</code>
              <code> {"{{metric_type}}"}</code>
              <code> {"{{bucket_time}}"}</code>
              <code> {"{{threshold_mb}}"}</code>
              <code> {"{{actual_mb}}"}</code>
              <code> {"{{alert_key}}"}</code>
            </p>
          </div>
          {props.webhookPreview ? (
            <div className="card">
              <div className="card-header">
                <strong>渲染预览</strong>
              </div>
              <p className="card-meta">URL</p>
              <pre className="code-block">{props.webhookPreview.url || "-"}</pre>
              <p className="card-meta">Headers</p>
              <pre className="code-block">{props.webhookPreview.headersText || "{}"}</pre>
              <p className="card-meta">Body</p>
              <pre className="code-block">{props.webhookPreview.bodyText || "-"}</pre>
            </div>
          ) : null}
          <div className="action-row">
            <button className="secondary-button" disabled={props.busy} onClick={props.onTestWebhook} type="button">
              测试 Webhook
            </button>
            <button className="primary-button" disabled={props.busy || props.webhookSaved} type="submit">
              保存 Webhook
            </button>
          </div>
        </form>
      </section>

      <section className="panel">
        <h3 className="panel-title">Telegram 通知</h3>
        <form className="form-grid" onSubmit={props.onSaveTelegram}>
          <label className="field checkbox-field">
            <input
              checked={props.telegramForm.enabled}
              onChange={(event) => {
                props.onTelegramFormChange((current) => ({ ...current, enabled: event.target.checked }));
              }}
              type="checkbox"
            />
            <span>启用 Telegram</span>
          </label>
          <label className="field">
            <span>Bot Token</span>
            <input
              value={props.telegramForm.botToken}
              onChange={(event) => {
                props.onTelegramFormChange((current) => ({ ...current, botToken: event.target.value }));
              }}
              placeholder="仅保存时填写"
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
          <button className="primary-button" disabled={props.busy || props.telegramSaved} type="submit">
            保存 Telegram
          </button>
        </form>

        <div className="list-block">
          <h4>当前渠道状态</h4>
          {props.notificationChannels.map((channel) => (
            <article className="card" key={channel.channel_type}>
              <div className="card-header">
                <strong>{channel.channel_type}</strong>
                <span className={`status-badge ${channel.enabled ? "ok" : "idle"}`}>
                  {channel.enabled ? "启用" : "停用"}
                </span>
              </div>
              <p className="card-meta">已配置：{channel.configured ? "是" : "否"}</p>
              {channel.url ? <p className="card-meta">URL：{channel.url}</p> : null}
              {channel.chat_id ? <p className="card-meta">Chat ID：{channel.chat_id}</p> : null}
              {channel.token_masked ? <p className="card-meta">Token：{channel.token_masked}</p> : null}
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
