import re
import json
from collections import defaultdict, deque

import requests
from matplotlib import pyplot as plt
from tabulate import tabulate

K = 3
END_MARKER = '$'

def send_req_cnf(cfg: str) -> str:
    url = 'http://localhost:8080/cnf'  # 45.132.19.101
    headers = {'Content-Type': 'application/json'}

    productions = cfg.strip().split('\n')
    data = {'productions': productions}

    response = requests.post(url, json=data, headers=headers)

    if response.status_code == 200:
        print("Ответ сервера (/cnf):")
        result = response.json()
        cnf = '\n'.join(result['productions'])
        print(cnf)
        return cnf
    else:
        print(f"Ошибка: {response.status_code}, {response.text}")
        return None

def send_req_test(req: dict):
    url = "http://localhost:8080/test"
    headers = {'Content-Type': 'application/json'}

    response = requests.post(url, json=req, headers=headers)
    if response.status_code == 200:
        try:
            data = response.json()
            return data
        except json.JSONDecodeError:
            print("Ошибка декодирования JSON:")
            print(response.text)
            return None
    else:
        print(f"Ошибка: {response.status_code}, {response.text}")
        return None


def tokenize_rhs(rhs):
    tokens = []
    pos = 0
    while pos < len(rhs):
        if rhs[pos].isspace():
            pos += 1
            continue
        elif rhs[pos] == '[':
            end = rhs.find(']', pos)
            if end == -1:
                raise ValueError(f"Unmatched '[' in RHS: {rhs}")
            tokens.append(rhs[pos:end+1])
            pos = end + 1
        elif rhs[pos].isupper():
            match = re.match(r'[A-Z][A-Za-z0-9_]*', rhs[pos:])
            if match:
                tokens.append(match.group())
                pos += len(match.group())
            else:
                raise ValueError(f"Invalid NonTerminal starting at position {pos} in RHS: {rhs}")
        elif rhs[pos] == 'ε':
            tokens.append('ε')
            pos += 1
        else:
            tokens.append(rhs[pos])
            pos += 1
    return tokens


def parse_grammar(grammar_str: str):
    lines = [line.strip() for line in grammar_str.strip().split('\n') if line.strip()]
    grammar = defaultdict(list)
    nonterminals = set()
    terminals = set()

    rule_re = re.compile(r'^(\[[A-Za-z][A-Za-z0-9_]*\]|[A-Z][A-Za-z0-9_]*)\s*->\s*(.*)$')

    for line in lines:
        match = rule_re.match(line)
        if not match:
            raise ValueError(f"Неверный формат правила: {line}")
        lhs = match.group(1)
        rhs = match.group(2)

        nonterminals.add(lhs)

        # Разделяем альтернативы по '|'
        alternatives = [alt.strip() for alt in rhs.split('|')]
        for alternative in alternatives:
            if alternative == 'ε':
                rhs_symbols = ['ε']
            else:
                rhs_symbols = tokenize_rhs(alternative)

            grammar[lhs].append(rhs_symbols)

    for lhs in grammar:
        for production in grammar[lhs]:
            for symbol in production:
                if symbol == 'ε':
                    terminals.add(symbol)
                elif re.fullmatch(r'\[[A-Za-z][A-Za-z0-9_]*\]', symbol):
                    nonterminals.add(symbol)
                elif re.fullmatch(r'[A-Z][A-Za-z0-9_]*', symbol):
                    nonterminals.add(symbol)
                else:
                    terminals.add(symbol)

    if not grammar:
        raise ValueError("Грамматика пуста.")

    start_symbol = next(iter(grammar))
    return grammar, nonterminals, terminals, start_symbol


def collect_grammar(grammar: dict) -> str:
    new_gram = ""
    for nt, prod_list in grammar.items():
        prods_str = " | ".join([" ".join(prod) for prod in prod_list])
        new_gram += f"{nt} -> {prods_str}\n"
    return new_gram.strip() + '\n'

def compute_production_first_k(production, k, FIRST_map): # Возвращает все префиксы длины <= k, с которыми может начинаться вывод из данной production
    prefixes = set([()])
    for symbol in production:
        new_prefixes = set()

        if symbol == 'ε':
            new_prefixes.update(prefixes)
            new_prefixes.add(('ε',))
            prefixes = new_prefixes
            break

        if symbol in FIRST_map:
            expansions = FIRST_map[symbol] #все возможные префиксы (до k) для символа
        else:
            expansions = set([(symbol,)])

        # Расширяем уже накопленные префиксы
        for prefix in prefixes:
            if prefix == ('ε',):
                combined_prefix = ()
            else:
                combined_prefix = prefix
            if len(combined_prefix) >= k: # Если уже есть k символов в префиксе, не достаточно
                new_prefixes.add(combined_prefix[:k])
                continue

            for s in expansions:  # Приписываем все expansions
                if s == ('ε',):
                    if combined_prefix == ():
                        new_prefixes.add(('ε',))
                    else:
                        new_prefixes.add(combined_prefix)
                else:
                    combined = combined_prefix + s
                    if len(combined) > k:
                        combined = combined[:k]
                    new_prefixes.add(combined)

        prefixes = new_prefixes
        if all(len(p) >= k for p in prefixes if p != ('ε',)):  # Если у всех префиксов уже длина >= k, то готово, выходим
            break

    return set([p for p in prefixes if len(p) <= k or p == ('ε',)])

def compute_FIRST_k(grammar, terminals, k):
    FIRST = defaultdict(set)

    for terminal in terminals:
        FIRST[terminal].add((terminal,))
    FIRST[END_MARKER].add((END_MARKER,))
    changed = True
    while changed:  # Итерируемся, пока не перестанут появляться новые префиксы
        changed = False
        for nonterminal in grammar:
            for production in grammar[nonterminal]:
                production_first_k = compute_production_first_k(production, k, FIRST)
                before = len(FIRST[nonterminal])
                FIRST[nonterminal].update(production_first_k)
                after = len(FIRST[nonterminal])
                if after > before:
                    changed = True
    return FIRST

def first_of_sequence(sequence, k, FIRST_map):
    # берём возможные префиксы до k для всех символов по порядку
    if not sequence:
        return set([('ε',)])

    result = set([('ε',)])
    for symbol in sequence:
        new_result = set()
        for prefix in result:
            if prefix == ('ε',):
                combined_prefix = ()
            else:
                combined_prefix = prefix

            expansions = FIRST_map[symbol]
            for s in expansions:
                if s == ('ε',):
                    if combined_prefix == ():
                        new_result.add(('ε',))
                    else:
                        new_result.add(combined_prefix)
                else:
                    combined = combined_prefix + s
                    if len(combined) > k:
                        combined = combined[:k]
                    new_result.add(combined)
        result = new_result

    # Если все символы могут давать 'ε', то тоже добавляем 'ε'
    can_be_empty = all(('ε',) in FIRST_map[sym] for sym in sequence)
    if can_be_empty:
        result.add(('ε',))

    return set([p for p in result if len(p) <= k or p == ('ε',)])

def compute_FOLLOW_k(grammar, nonterminals, start_symbol, k, FIRST):
    FOLLOW = defaultdict(set)
    FOLLOW[start_symbol].add((END_MARKER,))

    changed = True
    while changed:
        changed = False
        for A in grammar:
            for production in grammar[A]:
                for i, B in enumerate(production):
                    if B in nonterminals:
                        # Берём символы после B (beta) (продукция вида A -> <aльфа>B<бета>
                        beta = production[i + 1:]
                        first_beta = first_of_sequence(beta, k, FIRST)

                        before = len(FOLLOW[B])
                        # Если в first_beta нет ε, добавляем сами префиксы
                        for fb in first_beta:
                            if fb != ('ε',):
                                FOLLOW[B].add(fb[:k])

                        # Если в first_beta есть ε или beta пустое, добавляем FOLLOW(A)
                        if ('ε',) in first_beta or not beta:
                            for fA in FOLLOW[A]:
                                FOLLOW[B].add(fA[:k])
                        after = len(FOLLOW[B])
                        if after > before:
                            changed = True
    return FOLLOW

def is_all_terminals_or_epsilon(s, nonterminals):
    for symbol in s:
        if symbol == 'ε' or symbol == END_MARKER:
            continue
        if symbol in nonterminals:
            return False
    return True

def expand_string_until_terminals(s, FIRST_map, k, nonterminals):
    result = set()
    queue = deque()
    queue.append(s)
    visited = set()

    while queue:
        current = queue.popleft()

        # Если мы уже видели такую строку, пропускаем её
        if current in visited:
            continue
        visited.add(current)

        # Если вся строка (кортеж) содержит только терминалы (либо ε/$),
        # то добавляем её в результат (с учётом ограничения в k)
        if is_all_terminals_or_epsilon(current, nonterminals):
            if current == ('ε',):
                result.add(('ε',))
            else:
                if len(current) <= k:
                    result.add(current)
                else:
                    result.add(current[:k])
            continue

        # Иначе ищем первый нетерминал и раскрываем его
        idx_nt = -1
        for idx, symbol in enumerate(current):
            if symbol in nonterminals:
                idx_nt = idx
                break

        # Если нет нетерминалов, добавляем и дальше идём
        if idx_nt == -1:
            if len(current) <= k:
                result.add(current)
            else:
                result.add(current[:k])
            continue

        # Расширяем нетерминал
        A = current[idx_nt]
        expansions = FIRST_map.get(A, set())

        for expansion in expansions:
            if expansion == ('ε',):
                new_str = current[:idx_nt] + current[idx_nt+1:]
            else:
                new_str = current[:idx_nt] + expansion + current[idx_nt+1:]

            if len(new_str) <= k:
                queue.append(new_str)
            else:
                queue.append(new_str[:k])

    return result

def clean_up_first_follow(FIRST, FOLLOW, grammar, nonterminals, k):
    FIRST_clean = defaultdict(set)
    FOLLOW_clean = defaultdict(set)

    for A in FIRST:
        for s in FIRST[A]:
            expansions = ( expand_string_until_terminals(s, FIRST, k, nonterminals)
                           if s != ('ε',)
                           else set([('ε',)]) )
            for exp in expansions:
                if exp == ('ε',):
                    FIRST_clean[A].add(('ε',))
                else:
                    FIRST_clean[A].add(exp[:k])

    for A in FOLLOW:
        for s in FOLLOW[A]:
            expansions = ( expand_string_until_terminals(s, FIRST, k, nonterminals)
                           if s != ('ε',)
                           else set([('ε',)]) )
            for exp in expansions:
                if exp == ('ε',):
                    FOLLOW_clean[A].add(('ε',))
                else:
                    FOLLOW_clean[A].add(exp[:k])

    return FIRST_clean, FOLLOW_clean

def lookahead_k(A, production, k, FIRST_map, FOLLOW_map):
    # Собираем все lookahead-префиксы до к: FIRST_k(production) + FOLLOW(A), если есть ε
    first_alpha = compute_production_first_k(production, k, FIRST_map)
    first_follow = set()
    for prefix in first_alpha:
        if prefix == ('ε',):
            first_follow |= FOLLOW_map[A]
        else:
            first_follow.add(prefix)
    return first_follow

def build_LLk_table(grammar, nonterminals, k, FIRST, FOLLOW):
    parse_table = defaultdict(list)
    for A in grammar:
        for production in grammar[A]:
            LA = lookahead_k(A, production, k, FIRST, FOLLOW)
            if len(LA) != 0:
                for seq in LA:
                    print("seq", seq)
                    if len(seq) != 0:
                        parse_table[(A, seq)].append(production)
    return parse_table

def simulate_pda(word, parse_table, start_symbol, k, terminals, max_depth=1900):
    input_buffer = list(word) + ['$']

    def parse(stack, buffer, log, visited, depth=0):
        if depth > max_depth:
            log.append(f"ОШИБКА: Превышен предел глубины ({max_depth}). Предполагаем зацикливание.")
            return False, log

        config = (tuple(stack), tuple(buffer)) # Проверка на зацикливание (стек+буфер уже встречались)
        if config in visited:
            log.append("ОШИБКА: Зацикливание, повторяем одну и ту же конфигурацию!")
            return False, log

        visited = visited.copy()
        visited.add(config)

        if not stack:
            if buffer == ['$']:
                log.append("УСПЕХ: входная строка полностью разобрана.")
                return True, log
            else:
                log.append(f"ОШИБКА: стек пуст, но во входе осталось {buffer}.")
                return False, log

        current_symbol = stack.pop()
        lookahead = buffer[:k]
        current_input = ' '.join(lookahead)

        log.append(f"Стек: {stack[::-1]}, вход: {' '.join(buffer)}, "
                   f"текущий символ: {current_symbol}, lookahead: '{current_input}'")

        # Если это терминал (или $)
        if current_symbol in terminals or current_symbol == '$':
            if buffer and current_symbol == buffer[0]:
                log.append(f"  Терминал '{current_symbol}' совпал. Удаляем из буфера.")
                buffer.pop(0)
                return parse(stack, buffer, log, visited, depth+1)
            else:
                expected = current_symbol
                actual = buffer[0] if buffer else '$'
                log.append(f"  ОШИБКА: ожидается '{expected}', а найдено '{actual}'.")
                return False, log

        # Иначе (нетерминал)
        for i in range(1, len(lookahead)+1):
            seq = tuple(lookahead[:i])
            productions = parse_table.get((current_symbol, seq), [])
            if productions:
                # Сортируем так, чтобы ε-продукции были в конце (не знаю, надо или нет - из-за одного кейса с k=3 при k=1)
                sorted_productions = sorted(productions, key=lambda p: p == ['ε'])
                for production in sorted_productions:
                    log_copy = log.copy()
                    log_copy.append(f"  Применяем правило: {current_symbol} -> {' '.join(production)}")

                    new_stack = stack.copy()
                    if production != ['ε']:
                        new_stack.extend(reversed(production))

                    new_buffer = buffer.copy()

                    success, new_log = parse(new_stack, new_buffer, log_copy, visited, depth + 1)
                    if success:
                        return True, new_log
                    else:
                        log.append(f"  Правило {current_symbol} -> {' '.join(production)} не подошло, пробуем другое.")

        log.append(f"  ОШИБКА: нет продукции для ({current_symbol}, '{current_input}').")
        return False, log

    initial_stack = [start_symbol]
    initial_log = []
    visited_states = set()

    return parse(initial_stack, input_buffer, initial_log, visited_states, depth=0)

def save_table_as_image(table_data, headers, filename="parse_table.png"):
    fig, ax = plt.subplots(figsize=(max(8, len(headers)*1.2), max(6, len(table_data)*0.5)))
    ax.axis('tight')
    ax.axis('off')
    tbl = ax.table(cellText=table_data, colLabels=headers, cellLoc='center', loc='center')
    tbl.auto_set_font_size(False)
    tbl.set_fontsize(8)
    tbl.scale(1.2, 1.2)

    plt.savefig(filename, bbox_inches='tight')
    plt.close()
    print(f"Таблица парсинга сохранена как {filename}")

def main():
    if K == 0 or K > 3:
        print("АЙ АЙ АЙ, K должен принадлежать множеству (0, 3]!")
        return
    with open("input.txt", "r", encoding="utf-8") as f:
        lines = [l.strip() for l in f.readlines() if l.strip()]

    mode_line = lines[0]
    if not mode_line.startswith("mode="):
        print("Ошибка: первая строка файла input.txt должна быть вида 'mode=fuzz' или 'mode=manual'.")
        return

    mode = mode_line.split("=", maxsplit=1)[1].strip()

    grammar_text = "\n".join(lines[1:])

    grammar, nonterminals, terminals, start_symbol = parse_grammar(grammar_text)

    original_grammar_text = collect_grammar(grammar)

    # (возвращает грамматику без левой рекурсии и с левой факторизацией)
    print("Грамматика (исходная) для отправки на фаззер:\n", original_grammar_text)
    new_gram = send_req_cnf(original_grammar_text)
    if not new_gram:
        print("Ошибка при получении ""чистой"" грамматики.")
        return

    # Парсим уже чистую грамматику
    grammar, nonterminals, terminals, start_symbol = parse_grammar(new_gram)

    print("\nГрамматика чистая от сервера:", dict(grammar))
    print("Нетерминалы:", nonterminals)
    print("Терминалы:", terminals)
    print("Стартовый символ:", start_symbol)

    FIRST_raw = compute_FIRST_k(grammar, terminals, K)
    FOLLOW_raw = compute_FOLLOW_k(grammar, nonterminals, start_symbol, K, FIRST_raw)

    FIRST_clean, FOLLOW_clean = clean_up_first_follow(FIRST_raw, FOLLOW_raw, grammar, nonterminals, K)

    print("FIRST:", dict(FIRST_clean))
    print("FOLLOW:", dict(FOLLOW_clean))

    parse_table = build_LLk_table(grammar, nonterminals, K, FIRST_clean, FOLLOW_clean)

    print("\nLL(k) Таблица Парсинга (исп. очищенные FIRST/FOLLOW):")

    lookaheads = set()
    for (A, seq) in parse_table.keys():
        if len(seq) <= K:
            la = ' '.join(seq)

            lookaheads.add(la)

    for nt in FOLLOW_clean:
        for seq in FOLLOW_clean[nt]:
            if len(seq) <= K:
                if ' '.join(seq)!= "":
                    print("добавляем лукэхед: ", ' '.join(seq))
                    lookaheads.add(' '.join(seq))

    lookaheads.add(END_MARKER)
    lookaheads.discard('ε')

    lookaheads_sorted = sorted(lookaheads, key=lambda x: (len(x.split()), x))

    header = ["Нетерминал / Лукэхед"] + lookaheads_sorted

    nonterminals_sorted = sorted(nonterminals)
    table = []
    for nt in nonterminals_sorted:
        row = [nt]
        for la in lookaheads_sorted:
            key = (nt, tuple(la.split()))
            productions = parse_table.get(key, [])
            if productions:
                production_str = " | ".join([" ".join(prod) for prod in productions])
                row.append(production_str)
            else:
                row.append("")
        table.append(row)

    print(tabulate(table, headers=header, tablefmt="grid"))

    save_table_as_image(table, header, filename="parse_table.png")

    if "fuzz" in mode:
        test_config = {
            "pos": 10,
            "neg": 10,
            "wprob": 0.3,
            "attempts": 100000,
            "max_steps": 8,
            "productions": original_grammar_text.strip().split('\n'),
        }

        print("\nЗапрашиваю у фаззера набор слов для проверки...\n")
        test_response = send_req_test(test_config)
        if not test_response or 'tests' not in test_response:
            print("Не удалось получить тестовые слова от фаззера.")
            return

        iterCount = 0

        while True:
            iterCount += 1
            print("ITER NUM = ", iterCount)

            test_words = test_response['tests']

            total = len(test_words)
            matches = 0
            mismatches = []

            print("\n=== РЕЗУЛЬТАТ ТЕСТИРОВАНИЯ (FUZZER) ===")

            for idx, item in enumerate(test_words, start=1):
                word = item['word']
                server_decision = item['in_language']

                local_decision, parse_log = simulate_pda(word, parse_table, start_symbol, K, terminals)

                print(f"\nТЕСТ {idx}. Слово: '{word}'")
                print(f"  Фаззер говорит accepted = {server_decision}")
                print(f"  Мой парсер говорит accepted = {local_decision}")
                for line in parse_log:
                     print("    " + line)

                if server_decision == local_decision:
                    matches += 1
                else:
                    mismatches.append((word, server_decision, local_decision))

            print("\n=== ИТОГО ===")
            print(f"Всего слов: {total}")
            print(f"Совпадений результатов: {matches}")
            print(f"Несовпадений: {total - matches}")


            if mismatches:
                print("\nСписок несовпадений (слово / ответ сервера / ответ парсера):")
                for (w, srv, loc) in mismatches:
                    print(f"  {w}: сервер={srv}, локально={loc}")

            if total - matches != 0 or mode == "fuzz_limit":
                break

    elif mode == "manual":
        print("\nПереходим в режим ручной проверки.")
        print("Чтобы выйти, введите 'quit'.\n")

        while True:
            word = input("Введите слово для проверки: ").strip()
            if word.lower() == "quit":
                print("Завершение режима ручной проверки...")
                break

            local_decision, parse_log = simulate_pda(word, parse_table, start_symbol, K, terminals)

            print(f"Слово: '{word}'")
            print(f"Результат: {'accepted' if local_decision else 'rejected'}")
            print("Лог разбора:")
            for line in parse_log:
                print("   ", line)
            print()

    else:
        print(f"Неизвестный режим: {mode}. Ожидается 'fuzz' или 'manual'.")


if __name__ == "__main__":
    main()
