import type { CollectNowResponse, TrafficSample } from "../types";
import type { MachineOption } from "../lib/app-types";
import { formatPeriodType, formatStatusText, formatTime, formatTrafficValue, machineDisplay } from "../lib/app-utils";
import { useI18n } from "../lib/i18n";

type SamplesPageProps = {
  selectedMachineID: number | null;
  machineOptions: MachineOption[];
  samples: TrafficSample[];
  collectResults: CollectNowResponse["results"];
  onSelectMachine: (machineID: number | null) => void;
  onCollectCurrentMachine: (machineID: number) => void;
};
export default function SamplesPage(props: SamplesPageProps) {
  const { language, t } = useI18n();
  const filteredSamples = props.samples.filter(
    (sample) => !props.selectedMachineID || sample.machine_id === props.selectedMachineID,
  );

  return (
    <section className="panel">
      <div className="panel-header-inline">
        <h3 className="panel-title">{t("samplesTitle")}</h3>
        <div className="header-actions">
          {props.selectedMachineID ? (
            <button
              className="secondary-button"
              onClick={() => props.onCollectCurrentMachine(props.selectedMachineID!)}
              type="button"
            >
              {t("samplesCollectCurrentMachine")}
            </button>
          ) : null}
          <select
            value={props.selectedMachineID ?? ""}
            onChange={(event) => props.onSelectMachine(event.target.value ? Number(event.target.value) : null)}
          >
            <option value="">{t("samplesAllMachines")}</option>
            {props.machineOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>{t("tabMachines")}</th>
              <th>{t("thresholdPeriod")}</th>
              <th>{t("samplesBucketTime")}</th>
              <th>{t("metricUpload")}</th>
              <th>{t("metricDownload")}</th>
              <th>{t("metricTotal")}</th>
              <th>{t("samplesCollectedAt")}</th>
            </tr>
          </thead>
          <tbody>
            {filteredSamples.map((sample) => {
              const machine = machineDisplay(props.machineOptions, sample.machine_id, language);

              return (
                <tr key={`${sample.id}-${sample.period_type}`}>
                  <td>
                    <div className="machine-cell">
                      <strong>{machine.primary}</strong>
                      {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                    </div>
                  </td>
                  <td>{formatPeriodType(sample.period_type, language)}</td>
                  <td>{formatTime(sample.bucket_time, language)}</td>
                  <td>{formatTrafficValue(sample.upload_mb)}</td>
                  <td>{formatTrafficValue(sample.download_mb)}</td>
                  <td>{formatTrafficValue(sample.total_mb)}</td>
                  <td>{formatTime(sample.collected_at, language)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {props.collectResults.length > 0 ? (
        <div className="list-block">
          <h4>{t("samplesRecentResults")}</h4>
          {props.collectResults.map((result) => {
            const machine = machineDisplay(props.machineOptions, result.machine_id, language);

            return (
              <article className="card" key={`${result.machine_id}-${result.status}`}>
                <div className="card-header">
                  <div className="machine-cell">
                    <strong>{machine.primary}</strong>
                    {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                  </div>
                  <span className={`status-badge ${result.status === "success" ? "ok" : "error"}`}>
                    {formatStatusText(result.status, language)}
                  </span>
                </div>
                <p className="card-meta">{t("samplesSampleCount", { count: result.sample_count })}</p>
                {result.error ? <p className="card-meta">{t("samplesError", { error: result.error })}</p> : null}
              </article>
            );
          })}
        </div>
      ) : null}
    </section>
  );
}
