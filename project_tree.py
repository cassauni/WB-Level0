import os
import sys
import argparse

def generate_tree(directory, prefix=""):
    """
    Рекурсивно проходит по директории, формирует список строк для отображения дерева
    и собирает пути ко всем файлам.
    """
    tree_lines = []
    files_list = []
    try:
        items = sorted(os.listdir(directory))
    except PermissionError:
        tree_lines.append(prefix + "└── " + "[Нет доступа]")
        return tree_lines, files_list

    for index, item in enumerate(items):
        path = os.path.join(directory, item)
        connector = "└── " if index == len(items) - 1 else "├── "
        if os.path.isdir(path):
            tree_lines.append(prefix + connector + item)
            sub_prefix = prefix + ("    " if index == len(items) - 1 else "│   ")
            sub_tree, sub_files = generate_tree(path, sub_prefix)
            tree_lines.extend(sub_tree)
            files_list.extend(sub_files)
        else:
            tree_lines.append(prefix + connector + item)
            files_list.append(path)
    return tree_lines, files_list

def main():
    parser = argparse.ArgumentParser(
        description="Скрипт для генерации структуры проекта и чтения файлов"
    )
    parser.add_argument("project_path", help="Путь к директории проекта")
    parser.add_argument("-o", "--output", default="output.txt", help="Имя выходного текстового файла")
    parser.add_argument(
        "--exclude-files",
        nargs='*',
        default=[],
        help="Список имён файлов, содержимое которых не будет добавлено"
    )
    parser.add_argument(
        "--exclude-dirs",
        nargs='*',
        default=[],
        help="Список имён директорий, файлы внутри которых не будут добавлены"
    )
    args = parser.parse_args()
    
    project_path = args.project_path
    if not os.path.isdir(project_path):
        print(f"Ошибка: {project_path} не является корректной директорией.")
        sys.exit(1)
    
    exclude_files = set(args.exclude_files)
    exclude_dirs = set(args.exclude_dirs)

    tree_lines, files_list = generate_tree(project_path)

    output_lines = []
    output_lines.append("Структура проекта:")
    output_lines.append("\n".join(tree_lines))
    output_lines.append("\nСодержимое файлов:")

    for file_path in files_list:
        relative_path = os.path.relpath(file_path, project_path)

        # Проверяем, нужно ли пропустить файл
        basename = os.path.basename(file_path)
        dir_parts = os.path.dirname(relative_path).split(os.sep)
        if basename in exclude_files or any(d in exclude_dirs for d in dir_parts):
            continue

        # Если не пропускаем — выводим заголовок и содержимое
        output_lines.append("\n" + "=" * 40)
        output_lines.append(f"Файл: {relative_path}")
        output_lines.append("-" * 40)

        try:
            with open(file_path, 'r', encoding='utf-8', errors='replace') as f:
                content = f.read()
        except Exception as e:
            content = f"Ошибка при чтении файла: {e}"
        output_lines.append(content)
        output_lines.append("=" * 40)

    # Записываем результат
    with open(args.output, "w", encoding="utf-8") as f:
        f.write("\n".join(output_lines))
    
    print(f"Результат сохранён в файле {args.output}")

if __name__ == "__main__":
    main()
