import type { FormEvent } from "react";
import type { ConnectionTestResponse, Machine, SSHKey } from "../types";
import type { MachineFormState } from "../lib/app-types";

type MachinesPageProps = {
  busy: boolean;
  editingMachineID: number | null;
  machineForm: MachineFormState;
  machineFormSaved: boolean;
  sshKeys: SSHKey[];
  machines: Machine[];
  connectionResults: Record<number, ConnectionTestResponse>;
  onMachineSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onResetMachineForm: () => void;
  onUpdateMachineForm: <Key extends keyof MachineFormState>(key: Key, value: MachineFormState[Key]) => void;
  onStartEditMachine: (machine: Machine) => void;
  onTestConnection: (id: number) => void | Promise<void>;
  onDeleteMachine: (id: number) => void | Promise<void>;
};

export default function MachinesPage(props: MachinesPageProps) {
  const {
    busy,
    editingMachineID,
    machineForm,
    machineFormSaved,
    sshKeys,
    machines,
    connectionResults,
    onMachineSubmit,
    onResetMachineForm,
    onUpdateMachineForm,
    onStartEditMachine,
    onTestConnection,
    onDeleteMachine,
  } = props;

  return (
    <div className="grid two-columns">
      <section className="panel">
        <div className="panel-header-inline">
          <h3 className="panel-title">{editingMachineID ? "编辑机器" : "新增机器"}</h3>
          {editingMachineID ? (
            <button className="secondary-button" onClick={onResetMachineForm} type="button">
              取消编辑
            </button>
          ) : null}
        </div>

        <form className="form-grid" onSubmit={onMachineSubmit}>
          <label className="field">
            <span>名称</span>
            <input value={machineForm.name} onChange={(event) => onUpdateMachineForm("name", event.target.value)} />
          </label>

          <label className="field">
            <span>主机</span>
            <input value={machineForm.host} onChange={(event) => onUpdateMachineForm("host", event.target.value)} />
          </label>

          <label className="field">
            <span>端口</span>
            <input value={machineForm.port} onChange={(event) => onUpdateMachineForm("port", event.target.value)} />
          </label>

          <label className="field">
            <span>SSH 用户</span>
            <input value={machineForm.sshUser} onChange={(event) => onUpdateMachineForm("sshUser", event.target.value)} />
          </label>

          <label className="field">
            <span>网卡</span>
            <input
              value={machineForm.networkInterface}
              onChange={(event) => onUpdateMachineForm("networkInterface", event.target.value)}
            />
          </label>

          <label className="field">
            <span>SSH Key</span>
            <select value={machineForm.sshKeyID} onChange={(event) => onUpdateMachineForm("sshKeyID", event.target.value)}>
              <option value="">请选择 SSH Key</option>
              {sshKeys.map((sshKey) => (
                <option key={sshKey.id} value={sshKey.id}>
                  {sshKey.name}
                </option>
              ))}
            </select>
          </label>

          <label className="field checkbox-field">
            <input
              checked={machineForm.collectEnabled}
              onChange={(event) => onUpdateMachineForm("collectEnabled", event.target.checked)}
              type="checkbox"
            />
            <span>启用采集</span>
          </label>

          <label className="field full-width">
            <span>备注</span>
            <textarea
              rows={3}
              value={machineForm.remark}
              onChange={(event) => onUpdateMachineForm("remark", event.target.value)}
            />
          </label>

          <button className="primary-button" disabled={busy || machineFormSaved} type="submit">
            {editingMachineID ? "保存修改" : "创建机器"}
          </button>
        </form>
      </section>

      <section className="panel">
        <h3 className="panel-title">机器列表</h3>
        <div className="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>名称</th>
                <th>主机</th>
                <th>网卡</th>
                <th>SSH Key</th>
                <th>采集</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {machines.map((machine) => (
                <tr key={machine.id}>
                  <td>{machine.name}</td>
                  <td>
                    {machine.host}:{machine.port}
                  </td>
                  <td>{machine.network_interface}</td>
                  <td>{machine.ssh_key_id}</td>
                  <td>{machine.collect_enabled ? "启用" : "停用"}</td>
                  <td>
                    <div className="action-row">
                      <button className="secondary-button" onClick={() => onStartEditMachine(machine)} type="button">
                        编辑
                      </button>
                      <button className="secondary-button" onClick={() => void onTestConnection(machine.id)} type="button">
                        测试
                      </button>
                      <button className="danger-button" onClick={() => void onDeleteMachine(machine.id)} type="button">
                        删除
                      </button>
                    </div>
                    {connectionResults[machine.id] ? (
                      <p className="card-meta">
                        测试结果：{connectionResults[machine.id].status}
                        {connectionResults[machine.id].vnstat_version
                          ? ` / ${connectionResults[machine.id].vnstat_version}`
                          : ""}
                      </p>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
