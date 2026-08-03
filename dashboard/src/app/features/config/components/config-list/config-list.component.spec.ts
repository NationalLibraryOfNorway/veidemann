import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {BehaviorSubject, Subject} from 'rxjs';
import {ConfigListComponent} from './config-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ConfigObject, ListDataSource} from '../../../../shared/models';

describe('ConfigListComponent', () => {
  let component: ConfigListComponent;
  let fixture: ComponentFixture<ConfigListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        ConfigListComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ConfigListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('clears list and container selection when a new query resets the data source', () => {
    const query = new BehaviorSubject('first');
    const rows = new Subject<ConfigObject>();
    const selections: ConfigObject[][] = [];
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: () => rows,
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    component.dataSource = dataSource;
    component.selectedChange.subscribe(selected => selections.push(selected));

    rows.next(new ConfigObject({id: 'one'}));
    component.onMasterCheckboxToggle(true);
    expect(component.selectedRows()).toHaveLength(1);

    query.next('second');

    expect(component.selectedRows()).toEqual([]);
    expect(selections.at(-1)).toEqual([]);
  });
});
