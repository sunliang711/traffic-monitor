import { useI18n } from "../lib/i18n";

type PageSizeSelectProps = {
  value: number;
  onChange: (pageSize: number) => void;
};

const pageSizeOptions = [10, 25, 50, 100];

export default function PageSizeSelect(props: PageSizeSelectProps) {
  const { t } = useI18n();

  return (
    <label className="page-size-control">
      <span>{t("pageSizeLabel")}</span>
      <select value={props.value} onChange={(event) => props.onChange(Number(event.target.value))}>
        {pageSizeOptions.map((pageSize) => (
          <option key={pageSize} value={pageSize}>
            {t("pageSizeValue", { size: pageSize })}
          </option>
        ))}
      </select>
    </label>
  );
}
