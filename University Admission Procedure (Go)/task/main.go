package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Student struct {
	FullName          string
	EntranceExamScore float64
	ExamScores        []ExamScore
	Departments       []string
}

type ExamScore struct {
	Subject string
	Score   float64
}

const admissionStages int = 3

func main() {
	// 1. Считываем квоту на факультет
	n := readQuota()

	// 2. Читаем и парсим файл абитуриентов
	students := readApplicants("applicants.txt")

	// 3. Определяем факультеты и инициализируем мап поступивших
	departments := []string{"Biotech", "Chemistry", "Engineering", "Mathematics", "Physics"}
	admitted := make(map[string][]Student)

	// 4. Делаем копию списка студентов — чтобы не мутировать оригинал
	remaining := make([]Student, len(students))
	copy(remaining, students)

	// 5. Распределяем студентов по факультетам волнами (всего 3)
	for stage := 0; stage < admissionStages; stage++ {
		processDistributionStage(stage, n, departments, admitted, &remaining)
	}

	// 6. Запись списка поступивших в файлы по факультетам
	writeAdmissions(departments, admitted)
}

// readQuota - получает квоту из stdin
func readQuota() int {
	var n int
	_, err := fmt.Scan(&n)
	if err != nil {
		log.Fatalf("wrong input: %v", err)
	}
	return n
}

// readApplicants — Читает и парсит файл, возвращает []Student
func readApplicants(filename string) []Student {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	subjects := []string{"Physics", "Chemistry", "Math", "ComputerScience"}

	lines := strings.Split(string(data), "\n")
	students := make([]Student, 0, len(lines))

	for _, applicant := range lines {
		parts := strings.Fields(applicant)
		fullName := fmt.Sprintf("%s %s", parts[0], parts[1])
		scores := parts[2 : 2+len(subjects)]
		entrance := parts[2+len(subjects)]

		examScores := make([]ExamScore, 0, len(subjects))
		for i, s := range scores {
			score, _ := strconv.ParseFloat(s, 64)
			examScores = append(examScores, ExamScore{
				Subject: subjects[i],
				Score:   score,
			})
		}
		entranceScore, _ := strconv.ParseFloat(entrance, 64)
		departments := parts[2+len(subjects)+1:]
		students = append(students, Student{
			fullName,
			entranceScore,
			examScores,
			departments,
		})
	}
	return students
}

// processDistributionStage — Обрабатывает одну волну распределения по факультетам
func processDistributionStage(stage, quota int, departments []string, admitted map[string][]Student, remaining *[]Student) {
	// Мап факультет: [абитуриенты] на текущей волне
	applicants := make(map[string][]Student)

	// Заполняем абитуриентов по приоритетам факультетов
	for _, student := range *remaining {
		dep := student.Departments[stage]
		applicants[dep] = append(applicants[dep], student)
	}

	// Для каждого факультета выбираем отличников по результатам экзаменов
	for _, dep := range departments {
		if len(applicants[dep]) == 0 {
			continue
		}
		// Сортировка по убыванию оценки за экзамен, затем по имени
		slices.SortFunc(applicants[dep], func(a, b Student) int {
			scoreA := getBestScore(a, dep)
			scoreB := getBestScore(b, dep)
			if scoreA != scoreB {
				if scoreA > scoreB {
					return -1
				}
				return 1
			}
			return strings.Compare(a.FullName, b.FullName)
		})

		// Считаем сколько еще осталось мест на факультете
		toAdmit := quota - len(admitted[dep])
		// Выбираем минимум желающих (например если мест еще осталось 3, но подался всего 1 или наоборот)
		numAdmit := min(toAdmit, len(applicants[dep]))
		// защита от нулевых значений
		if numAdmit > 0 {
			admitted[dep] = append(admitted[dep], applicants[dep][:numAdmit]...)
		}
	}

	// Создаем словарь тех, кто уже поступил
	accepted := make(map[string]struct{})
	for _, dep := range departments {
		for _, student := range admitted[dep] {
			accepted[student.FullName] = struct{}{}
		}
	}

	// Новый список студентов, которые остались без места на факультете
	tmp := (*remaining)[:0]

	for _, student := range *remaining {
		if _, ok := accepted[student.FullName]; !ok {
			tmp = append(tmp, student)
		}
	}
	*remaining = tmp
}

// printAdmissions — Выводит списки зачисленных по факультетам
func printAdmissions(departments []string, admitted map[string][]Student) {
	for _, dep := range departments {
		fmt.Println(dep)
		students := admitted[dep]
		slices.SortFunc(students, func(a, b Student) int {
			scoreA := getBestScore(a, dep)
			scoreB := getBestScore(b, dep)
			if scoreA != scoreB {
				if scoreA > scoreB {
					return -1
				}
				return 1
			}
			return strings.Compare(a.FullName, b.FullName)
		})
		for _, student := range students {
			fmt.Printf("%s %.1f\n", student.FullName, getExamMeanScore(student, dep))
		}
		fmt.Println()
	}
}

func writeAdmissions(departments []string, admitted map[string][]Student) {
	for _, dep := range departments {
		students := admitted[dep]
		slices.SortFunc(students, func(a, b Student) int {
			scoreA := getBestScore(a, dep)
			scoreB := getBestScore(b, dep)
			if scoreA != scoreB {
				if scoreA > scoreB {
					return -1
				}
				return 1
			}
			return strings.Compare(a.FullName, b.FullName)
		})

		file, err := os.Create(strings.ToLower(dep) + ".txt")
		if err != nil {
			log.Fatalf("failed to create file for %s: %v", dep, err)
		}
		for _, student := range students {
			fmt.Fprintf(file, "%s %.2f\n", student.FullName, getBestScore(student, dep))
		}
		file.Close()
	}
}

func getExamMeanScore(s Student, dep string) float64 {
	depToSubjects := map[string][]string{
		"Physics":     {"Physics", "Math"},
		"Chemistry":   {"Chemistry"},
		"Mathematics": {"Math"},
		"Engineering": {"ComputerScience", "Math"},
		"Biotech":     {"Chemistry", "Physics"},
	}

	subjects := depToSubjects[dep]
	var sum float64

	for _, subj := range subjects {
		for _, score := range s.ExamScores {
			if score.Subject == subj {
				sum += score.Score
			}
		}
	}
	return sum / float64(len(subjects))
}

// getBestScore - берет либо результат вступительного экзамена либо результат профильных экзаменов
func getBestScore(s Student, dep string) float64 {
	meanScore := getExamMeanScore(s, dep)
	entranceScore := s.EntranceExamScore
	if meanScore > entranceScore {
		return meanScore
	}
	return entranceScore
}
