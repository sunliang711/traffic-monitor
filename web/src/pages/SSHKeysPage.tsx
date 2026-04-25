import type { FormEvent } from "react";
import type { SSHKey } from "../types";
import type { SSHKeyGenerateState, SSHKeyImportState } from "../lib/app-types";

type SSHKeysPageProps = {
  busy: boolean;
  sshKeys: SSHKey[];
  sshImportForm: SSHKeyImportState;
  sshGenerateForm: SSHKeyGenerateState;
  setSSHImportForm: React.Dispatch<React.SetStateAction<SSHKeyImportState>>;
  setSSHGenerateForm: React.Dispatch<React.SetStateAction<SSHKeyGenerateState>>;
  onImportSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onGenerateSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onDeleteSSHKey: (id: number) => void | Promise<void>;
};

export default function SSHKeysPage(props: SSHKeysPageProps) {
  return (
    <div className="grid two-columns">
      <section className="panel">
        <h3 className="panel-title">导入已有 SSH Key</h3>
        <form className="form-grid" onSubmit={props.onImportSubmit}>
          <label className="field">
            <span>名称</span>
            <input
              value={props.sshImportForm.name}
              onChange={(event) =>
                props.setSSHImportForm((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
              placeholder="例如：prod-root"
            />
          </label>
          <label className="field">
            <span>私钥</span>
            <textarea
              rows={10}
              value={props.sshImportForm.privateKey}
              onChange={(event) =>
                props.setSSHImportForm((current) => ({
                  ...current,
                  privateKey: event.target.value,
                }))
              }
              placeholder="粘贴 OpenSSH 私钥"
            />
          </label>
          <button
            className="primary-button"
            disabled={props.busy || !props.sshImportForm.name.trim() || !props.sshImportForm.privateKey.trim()}
            type="submit"
          >
            导入
          </button>
        </form>
      </section>

      <section className="panel">
        <h3 className="panel-title">生成新 Keypair</h3>
        <form className="form-grid" onSubmit={props.onGenerateSubmit}>
          <label className="field">
            <span>名称</span>
            <input
              value={props.sshGenerateForm.name}
              onChange={(event) => props.setSSHGenerateForm({ name: event.target.value })}
              placeholder="例如：ops-generated"
            />
          </label>
          <button className="primary-button" disabled={props.busy} type="submit">
            生成
          </button>
        </form>

        <div className="list-block">
          <h4>SSH Key 列表</h4>
          {props.sshKeys.map((sshKey) => (
            <article className="card" key={sshKey.id}>
              <div className="card-header">
                <strong>{sshKey.name}</strong>
                <button className="danger-button" onClick={() => void props.onDeleteSSHKey(sshKey.id)} type="button">
                  删除
                </button>
              </div>
              <p className="card-meta">
                类型：{sshKey.key_type} / 来源：{sshKey.source_type}
              </p>
              <p className="card-meta">指纹：{sshKey.fingerprint}</p>
              <pre className="code-block">{sshKey.public_key}</pre>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
